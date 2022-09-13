package amqp

import (
	"reflect"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"github.com/go-foreman/foreman/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	delay          = time.Second * 3 // reconnect after delay seconds
	reconnectCount = 20
)

// Dial wrap amqp.Dial, dial and get a reconnect connection
func Dial(url string, autoReconnect bool, logger log.Logger) (UnderlyingConnection, error) {
	var connPtr UnderlyingConnection

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	connPtr = conn

	if autoReconnect {
		go func() {
			for {
				var reconnectedCount uint

				reason, ok := <-conn.NotifyClose(make(chan *amqp.Error))
				// exit this goroutine if closed by developer
				if !ok {
					logger.Log(log.InfoLevel, "connection closed explicitly")
					break
				}

				logger.Logf(log.WarnLevel, "connection closed, reason: %v", reason)

				// reconnect if not closed by developer
				for {
					// wait for reconnect
					time.Sleep(delay)

					if reconnectedCount > reconnectCount {
						logger.Logf(log.FatalLevel, "reached limit of reconnects %d", reconnectCount)
					}
					reconnectedCount++

					conn, err := amqp.Dial(url)
					if err == nil {
						reflect.ValueOf(connPtr).Elem().Set(reflect.ValueOf(conn).Elem())
						logger.Log(log.InfoLevel, "successfully reconnected amqp.Connection")
						break
					}

					logger.Logf(log.ErrorLevel, "reconnect failed, err: %v", err)
				}
			}
		}()
	}

	return connPtr, nil
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
