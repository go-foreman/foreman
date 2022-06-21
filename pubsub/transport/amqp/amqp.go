package amqp

import (
	"time"

	"github.com/go-foreman/foreman/log"

	"context"
	"sync"

	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

func NewTransport(url string, logger log.Logger) transport.Transport {
	return &amqpTransport{
		url:    url,
		logger: logger,
	}
}

type amqpTransport struct {
	url               string
	connection        *Connection
	publishingChannel *Channel
	logger            log.Logger
}

func (t *amqpTransport) Connect(ctx context.Context) error {
	conn, err := Dial(t.url, t.logger)
	if err != nil {
		return errors.WithStack(err)
	}

	publishingChannel, err := conn.Channel()

	if err != nil {
		return errors.WithStack(err)
	}

	t.connection = conn
	t.publishingChannel = publishingChannel

	return nil
}

// CreateTopic creates an exchange in amqp. Allows options are: durable, autoDelete, internal, noWait.
func (t *amqpTransport) CreateTopic(ctx context.Context, topic transport.Topic) error {
	if err := t.checkConnection(); err != nil {
		return errors.WithStack(err)
	}

	amqpTopic, topicConv := topic.(amqpTopic)

	if !topicConv {
		return errors.Errorf("Supplied topic is not an instance of amqp.Topic")
	}

	if err := t.publishingChannel.ExchangeDeclare(
		amqpTopic.Name(),
		"topic",
		amqpTopic.durable,
		amqpTopic.autoDelete,
		amqpTopic.internal,
		amqpTopic.noWait,
		nil,
	); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (t *amqpTransport) CreateQueue(ctx context.Context, q transport.Queue, qbs ...transport.QueueBind) error {
	if err := t.checkConnection(); err != nil {
		return errors.WithStack(err)
	}

	queue, queueConv := q.(amqpQueue)

	if !queueConv {
		return errors.Errorf("Supplied Queue is not an instance of amqp.amqpQueue")
	}

	var queueBinds []amqpQueueBind

	for _, item := range qbs {
		queueBind, queueBindConv := item.(amqpQueueBind)

		if !queueBindConv {
			return errors.Errorf("One of supplied QueueBinds is not an instance of amqp.amqpQueueBind")
		}

		queueBinds = append(queueBinds, queueBind)
	}

	if _, err := t.publishingChannel.QueueDeclare(
		queue.Name(),
		queue.durable,
		queue.autoDelete,
		queue.exclusive,
		queue.noWait,
		nil,
	); err != nil {
		return errors.WithStack(err)
	}

	for _, qb := range queueBinds {
		if err := t.publishingChannel.QueueBind(
			queue.Name(),
			qb.BindingKey(),
			qb.DestinationTopic(),
			qb.noWait,
			nil,
		); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func (t *amqpTransport) Send(ctx context.Context, outboundPkg transport.OutboundPkg, options ...transport.SendOpts) error {
	if err := t.checkConnection(); err != nil {
		return errors.WithStack(err)
	}

	sendOptions := &SendOptions{}

	for _, opt := range options {
		if err := opt(sendOptions); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := t.publishingChannel.Publish(
		outboundPkg.Destination().DestinationTopic,
		outboundPkg.Destination().RoutingKey,
		sendOptions.Mandatory,
		sendOptions.Immediate,
		amqp.Publishing{
			Headers:     outboundPkg.Headers(),
			ContentType: outboundPkg.ContentType(),
			Body:        outboundPkg.Payload(),
		},
	); err != nil {
		return errors.Wrap(err, "sending out pkg")
	}

	return nil
}

func (t *amqpTransport) Consume(ctx context.Context, queues []transport.Queue, options ...transport.ConsumeOpts) (<-chan transport.IncomingPkg, error) {
	if err := t.checkConnection(); err != nil {
		return nil, errors.WithStack(err)
	}

	consumingChannel, err := t.connection.Channel()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	consumeOptions := &ConsumeOptions{}

	for _, opt := range options {
		if err := opt(consumeOptions); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	if consumeOptions.PrefetchCount > 0 {
		if err := consumingChannel.Qos(int(consumeOptions.PrefetchCount), 0, false); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	income := make(chan transport.IncomingPkg)

	consumersWait := &sync.WaitGroup{}

	consumersCtx, cancelConsumers := context.WithCancel(ctx) //nolint:govet

	for _, q := range queues {
		consumingCh, err := consumingChannel.Consume(
			q.Name(),
			q.Name(),
			false,
			consumeOptions.Exclusive,
			consumeOptions.NoLocal,
			consumeOptions.NoWait,
			nil,
		)

		if err != nil {
			cancelConsumers() // this will shut down all goroutines previously created in this loop
			return nil, errors.Wrapf(err, "consuming %s", q.Name())
		}

		consumersWait.Add(1)

		go func(consumersCtx context.Context, queue transport.Queue, deliveries <-chan amqp.Delivery) {
			defer consumersWait.Done()

			defer func() {
				t.logger.Logf(log.InfoLevel, "canceling consumer %s", queue.Name())
				if err := consumingChannel.Cancel(queue.Name(), true); err != nil {
					t.logger.Logf(log.ErrorLevel, "error canceling consumer %s. %s", queue.Name(), err)
				} else {
					t.logger.Logf(log.InfoLevel, "canceled consumer %s", queue.Name())
				}
			}()

			for {
				select {
				case msg, open := <-deliveries:
					if !open {
						t.logger.Logf(log.WarnLevel, "Amqp consumer closed channel for queue %s", queue.Name())
						return
					}

					income <- &inAmqpPkg{origin: queue.Name(), receivedAt: time.Now(), delivery: msg}
				case <-consumersCtx.Done():
					t.logger.Logf(log.WarnLevel, "Canceled context. Stopped consuming queue %s", queue.Name())
					return
				}
			}
		}(consumersCtx, q, consumingCh)
	}

	go func() {
		consumersWait.Wait()
		close(income)

		if err := consumingChannel.Close(); err != nil {
			t.logger.Logf(log.ErrorLevel, "error closing amqp channel. %s", err)
		} else {
			t.logger.Log(log.InfoLevel, "closed consumer channel")
		}
	}()

	return income, nil //nolint:govet
}

func (t *amqpTransport) Disconnect(ctx context.Context) error {
	if t.connection == nil || t.publishingChannel == nil {
		return nil
	}

	if err := t.publishingChannel.Close(); err != nil {
		return errors.Wrap(err, "error closing publishing channel")
	}

	if err := t.connection.Close(); err != nil {
		return errors.Wrap(err, "error closing connection")
	}

	return nil
}

func (t *amqpTransport) checkConnection() error {
	if t.connection == nil {
		return errors.Errorf("Connection wasn't established. Use transport.Connect first")
	}

	return nil
}
