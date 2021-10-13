package amqp

import (
	"github.com/go-foreman/foreman/log"

	"context"
	"sync"

	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
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
	consumingChannel  *Channel
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

	consumingChannel, err := conn.Channel()

	if err != nil {
		return errors.WithStack(err)
	}

	t.connection = conn
	t.publishingChannel = publishingChannel
	t.consumingChannel = consumingChannel

	return nil
}

func (t *amqpTransport) CreateTopic(ctx context.Context, topic transport.Topic) error {
	if err := t.checkConnection(); err != nil {
		return errors.WithStack(err)
	}

	amqpTopic, topicConv := topic.(amqpTopic)

	if !topicConv {
		return errors.Errorf("Supplied topic is not an instance of amqp.Topic")
	}

	err := t.publishingChannel.ExchangeDeclare(
		amqpTopic.Name(),
		"topic",
		amqpTopic.durable,
		amqpTopic.autoDelete,
		amqpTopic.internal,
		amqpTopic.noWait,
		nil,
	)
	if err != nil {
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

	_, err := t.publishingChannel.QueueDeclare(
		queue.Name(),
		queue.durable,
		queue.autoDelete,
		queue.exclusive,
		queue.noWait,
		nil,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, qb := range queueBinds {
		err := t.publishingChannel.QueueBind(
			queue.Name(),
			qb.BindingKey(),
			qb.DestinationTopic(),
			qb.noWait,
			nil,
		)
		if err != nil {
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

	err := t.publishingChannel.Publish(
		outboundPkg.Destination().DestinationTopic,
		outboundPkg.Destination().RoutingKey,
		sendOptions.Mandatory,
		sendOptions.Immediate,
		amqp.Publishing{
			Headers:     outboundPkg.Headers(),
			ContentType: outboundPkg.ContentType(),
			Body:        outboundPkg.Payload(),
		},
	)
	if err != nil {
		return errors.Wrap(err, "sending out pkg")
	}

	return nil
}

func (t *amqpTransport) Consume(ctx context.Context, queues []transport.Queue, options ...transport.ConsumeOpts) (<-chan transport.IncomingPkg, error) {
	if err := t.checkConnection(); err != nil {
		return nil, errors.WithStack(err)
	}

	consumeOptions := &ConsumeOptions{}

	for _, opt := range options {
		if err := opt(consumeOptions); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	if consumeOptions.PrefetchCount > 0 {
		if err := t.consumingChannel.Qos(consumeOptions.PrefetchCount, 0, false); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	income := make(chan transport.IncomingPkg)

	consumersWait := &sync.WaitGroup{}

	for _, q := range queues {
		consumersWait.Add(1)
		go func(queue transport.Queue) {
			defer consumersWait.Done()

			defer func() {
				t.logger.Logf(log.InfoLevel, "canceling consumer %s", queue.Name())
				if err := t.consumingChannel.Cancel(queue.Name(), true); err != nil {
					t.logger.Logf(log.ErrorLevel, "error canceling consumer %s", err)
				}
				t.logger.Logf(log.InfoLevel, "canceled consumer %s", queue.Name())
			}()

			msgs, err := t.consumingChannel.Consume(
				queue.Name(),
				queue.Name(),
				false,
				consumeOptions.Exclusive,
				consumeOptions.NoLocal,
				consumeOptions.NoWait,
				nil,
			)

			if err != nil {
				t.logger.Log(log.ErrorLevel, err)
				return
			}

			for {
				select {
				case msg, open := <-msgs:
					if !open {
						t.logger.Logf(log.WarnLevel, "Amqp consumer closed channel for queue %s", queue.Name())
						return
					}

					income <- transport.NewAmqpIncomingPackage(msg, queue.Name())
				case <-ctx.Done():
					t.logger.Logf(log.WarnLevel, "Canceled context. Stopped consuming queue %s", queue.Name())
					return
				}
			}
		}(q)
	}

	go func() {
		consumersWait.Wait()
		close(income)
		t.logger.Logf(log.InfoLevel, "closed consumer channel")
	}()

	return income, nil
}

func (t *amqpTransport) Disconnect(ctx context.Context) error {
	if t.connection == nil || t.publishingChannel == nil || t.consumingChannel == nil {
		return nil
	}

	if err := t.publishingChannel.Close(); err != nil {
		return errors.Wrap(err, "error closing publishing channel")
	}

	if err := t.consumingChannel.Close(); err != nil {
		return errors.Wrap(err, "error closing consuming channel")
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
