package amqp

import (
	"sync/atomic"
	"time"

	"github.com/go-foreman/foreman/log"
	"github.com/streadway/amqp"
)

const (
	delay          = 3 // reconnect after delay seconds
	reconnectCount = 10
)

type Connection struct {
	logger log.Logger
	*amqp.Connection
}

// Channel wrap amqp.Connection.Channel, get a auto reconnect channel
func (c *Connection) Channel() (*Channel, error) {
	ch, err := c.Connection.Channel()
	if err != nil {
		return nil, err
	}

	channel := &Channel{
		Channel: ch,
		logger:  c.logger,
	}

	go func() {
		for {
			reason, ok := <-channel.Channel.NotifyClose(make(chan *amqp.Error))
			// exit this goroutine if closed by developer
			if !ok || channel.IsClosed() {
				c.logger.Log(log.WarnLevel, "channel closed")
				channel.Close() // close again, ensure closed flag set when connection closed
				break
			}
			c.logger.Logf(log.WarnLevel, "channel closed, reason: %v", reason)

			// reconnect if not closed by developer
			for {
				// wait 1s for connection reconnect
				time.Sleep(delay * time.Second)

				ch, err := c.Connection.Channel()
				if err == nil {
					c.logger.Log(log.InfoLevel, "channel recreation succeed")
					channel.Channel = ch
					break
				}

				c.logger.Logf(log.ErrorLevel, "channel recreate failed, err: %v", err)
			}
		}

	}()

	return channel, nil
}

// Channel amqp.Channel wapper
type Channel struct {
	logger log.Logger
	*amqp.Channel
	closed int32
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

	return ch.Channel.Close()
}

// Consume warp amqp.Channel.Consume, the returned delivery will end only when channel closed by developer
func (ch *Channel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	deliveries := make(chan amqp.Delivery)

	var reconnectedCount uint

	go func() {
		for {
			d, err := ch.Channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
			if err != nil {
				ch.logger.Logf(log.ErrorLevel, "consume failed, err: %v", err)
				time.Sleep(delay * time.Second)
				if reconnectedCount > reconnectCount {
					ch.logger.Logf(log.FatalLevel, "Reached limit of reconnects %d", reconnectCount)
				}
				reconnectedCount++
				continue
			}

			for msg := range d {
				deliveries <- msg
			}

			// sleep before IsClose call. closed flag may not set before sleep.
			time.Sleep(delay * time.Second)

			if ch.IsClosed() {
				break
			}
		}
	}()

	return deliveries, nil
}

// Dial wrap amqp.Dial, dial and get a reconnect connection
func Dial(url string, logger log.Logger) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	connection := &Connection{
		Connection: conn,
		logger:     logger,
	}

	var reconnectedCount uint

	go func() {
		for {
			reason, ok := <-connection.Connection.NotifyClose(make(chan *amqp.Error))
			// exit this goroutine if closed by developer
			if !ok {
				logger.Log(log.InfoLevel, "connection closed")
				break
			}
			logger.Logf(log.WarnLevel, "connection closed, reason: %v", reason)

			// reconnect if not closed by developer
			for {
				// wait 1s for reconnect
				time.Sleep(delay * time.Second)

				if reconnectedCount > reconnectCount {
					logger.Logf(log.FatalLevel, "Reached limit of reconnects %d", reconnectCount)
				}
				reconnectedCount++

				conn, err := amqp.Dial(url)
				if err == nil {
					connection.Connection = conn
					logger.Log(log.InfoLevel, "reconnect success")
					break
				}

				logger.Logf(log.ErrorLevel, "reconnect failed, err: %v", err)
			}
		}
	}()

	return connection, nil
}
