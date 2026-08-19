package amqp

import (
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/go-foreman/foreman/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	delay          = time.Second * 3 // reconnect after delay seconds
	reconnectCount = 20
	locale         = "en_US" // what amqp.Dial passes to amqp.DialConfig
)

// Dial wraps amqp.DialConfig, dial and get a reconnect connection.
//
// With autoReconnect the driver's own recovery is enabled: it redials the connection,
// reopens its channels, redeclares the recorded topology (exchanges, queues, bindings)
// and re-subscribes active consumers onto the very same delivery channels their callers
// already hold. Closing the connection stops recovery, so nothing is left running behind
// a shut-down application.
func Dial(url string, autoReconnect bool, logger log.Logger) (UnderlyingConnection, error) {
	config := amqp.Config{Locale: locale}

	if autoReconnect {
		config.Recovery = &amqp.Recovery{
			ReconnectionConfig: &amqp.ReconnectionConfig{
				MaxRetryCount: reconnectCount,
				RetryInterval: delay,
			},
			ConnectionRecovery: &loggingRecovery{logger: logger},
		}
	}

	conn, err := amqp.DialConfig(url, config)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// loggingRecovery reports connection and channel losses through foreman's logger and
// leaves the recovery itself to the driver's default implementation.
type loggingRecovery struct {
	logger   log.Logger
	delegate amqp.DefaultConnectionRecovery
}

func (r *loggingRecovery) OnConnectionClose(conn *amqp.Connection, err *amqp.Error) {
	r.logger.Logf(log.WarnLevel, "connection closed, reason: %v", err)
	r.delegate.OnConnectionClose(conn, err)
}

func (r *loggingRecovery) OnChannelClose(ch *amqp.Channel, err *amqp.Error) {
	r.logger.Logf(log.WarnLevel, "channel closed, reason: %v", err)
	r.delegate.OnChannelClose(ch, err)
}

// selfRecovering reports whether the driver takes care of reconnecting this connection
// and everything opened on it, making foreman's own supervision unnecessary.
func selfRecovering(conn UnderlyingConnection) bool {
	recoverable, ok := conn.(interface{ IsRecoveryEnabled() bool })
	return ok && recoverable.IsRecoveryEnabled()
}

type Connection struct {
	logger log.Logger
	//underlyingConn and Connection have to point to the same connection at all times
	underlyingConn      UnderlyingConnection
	chReconnectionDelay time.Duration
}

func NewReconnectConnection(logger log.Logger, underlyingConn UnderlyingConnection, chReconnectionDelay time.Duration) *Connection {

	return &Connection{
		logger:              logger,
		underlyingConn:      underlyingConn,
		chReconnectionDelay: chReconnectionDelay,
	}
}

func (c *Connection) Close() error {
	return c.underlyingConn.Close()
}

func (c *Connection) IsClosed() bool {
	return c.underlyingConn.IsClosed()
}

// Channel wrap amqp.Connection.Channel, get a auto reconnect channel
func (c *Connection) Channel() (AmqpChannel, error) {
	ch, err := c.underlyingConn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "creating channel")
	}

	channel := &Channel{
		AmqpChannel:              ch,
		logger:                   c.logger,
		consumeReconnectionDelay: c.chReconnectionDelay,
		selfRecovering:           selfRecovering(c.underlyingConn),
	}

	// A driver-recovered channel reopens itself and comes back with its topology and
	// consumers intact, so watching it here would only race with that recovery.
	if channel.selfRecovering {
		return channel, nil
	}

	go func() {
		for {
			reason, ok := <-channel.NotifyClose(make(chan *amqp.Error))
			// exit this goroutine if closed by developer
			if !ok || channel.IsClosed() {
				c.logger.Log(log.WarnLevel, "channel closed")
				// close again, ensure closed flag set when connection closed
				if err := channel.Close(); err != nil {
					c.logger.Logf(log.ErrorLevel, "error closing channel %s", err)
				}
				break
			}
			c.logger.Logf(log.WarnLevel, "channel closed, reason: %v", reason)

			// reconnect if not closed by developer
			for {
				// wait 3s for connection reconnect
				time.Sleep(c.chReconnectionDelay)

				//@todo here a panic happens panic: send on closed channel
				// How to reproduce:
				// 1. connect
				// 2. docker restart rabbitmq
				// 3. wait for successful reconnection
				// 4. docker restart rabbitmq
				// 5. the issue appears.
				ch, err = c.underlyingConn.Channel()
				if err == nil {
					channel.AmqpChannel = ch
					break
				}

				c.logger.Logf(log.ErrorLevel, "channel recreate failed, err: %v", err)
			}
		}

	}()

	return channel, nil
}

// Channel amqp.Channel wrapper
type Channel struct {
	AmqpChannel
	closed                   int32
	logger                   log.Logger
	consumeReconnectionDelay time.Duration
	selfRecovering           bool
}

// IsClosed indicate closed by developer
func (ch *Channel) IsClosed() bool {
	return atomic.LoadInt32(&ch.closed) == 1
}

// Close ensure closed flag set
func (ch *Channel) Close() error {
	if ch.IsClosed() {
		return amqp.ErrClosed
	}

	atomic.StoreInt32(&ch.closed, 1)
	return ch.AmqpChannel.Close()
}

// Consume warp amqp.Channel.Consume, the returned delivery will end only when channel closed by developer
func (ch *Channel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	// The driver re-subscribes a recovered consumer onto the same delivery channel it
	// returned here, which stays open across reconnects and closes only once the channel
	// itself is closed — exactly the contract this wrapper exists to provide.
	if ch.selfRecovering {
		return ch.AmqpChannel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	}

	deliveries := make(chan amqp.Delivery)

	var reconnectedCount uint

	go func() {
		defer close(deliveries)
		for {
			d, err := ch.AmqpChannel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
			if err != nil {
				ch.logger.Logf(log.ErrorLevel, "consume failed, err: %v", err)
				time.Sleep(ch.consumeReconnectionDelay)

				if reconnectedCount > reconnectCount {
					ch.logger.Logf(log.ErrorLevel, "Reached limit of reconnects %d", reconnectCount)
					break
				}

				reconnectedCount++
				ch.logger.Logf(log.DebugLevel, "retrying to reconnect consumer %s", consumer)

				continue
			}

			ch.logger.Logf(log.DebugLevel, "started consuming %s", consumer)

			for msg := range d {
				deliveries <- msg
			}

			// sleep before IsClose call. closed flag may not set before sleep.
			time.Sleep(ch.consumeReconnectionDelay)

			if ch.IsClosed() {
				break
			}
		}
	}()

	return deliveries, nil
}
