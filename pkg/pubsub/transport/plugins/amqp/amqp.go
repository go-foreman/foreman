package amqp

import (
	log "github.com/kopaygorodsky/brigadier/pkg/log"

	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"sync"
)

func NewTransport(url string, logger log.Logger) transport.Transport {
	return &amqpTransport{
		url:    url,
		logger: logger,
	}
}

type amqpTransport struct {
	url               string
	connection        *amqp.Connection
	publishingChannel *amqp.Channel
	logger            log.Logger
	consumingChannel  *amqp.Channel
}

func (t *amqpTransport) Connect(ctx context.Context) error {
	conn, err := amqp.Dial(t.url)
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

func (t *amqpTransport) Send(ctx context.Context, outboundPkg pkg.OutboundPkg, options ...transport.SendOpts) error {
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
		return errors.WithStack(err)
	}

	return nil
}

func (t *amqpTransport) Consume(ctx context.Context, queues []transport.Queue, options ...transport.ConsumeOpts) (<-chan pkg.IncomingPkg, error) {
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

	income := make(chan pkg.IncomingPkg)

	consumersWait := sync.WaitGroup{}

	for _, q := range queues {
		consumersWait.Add(1)
		go func(queue transport.Queue) {
			defer consumersWait.Done()

			//if err := ch.Qos(1, 0, true); err != nil {
			//	t.logger.Log(log.ErrorLevel, err)
			//	return
			//}

			defer func() {
				if err := t.consumingChannel.Cancel(queue.Name(), false); err != nil {
					t.logger.Logf(log.ErrorLevel, "error canceling consumer %s", err)
				}
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

					income <- pkg.NewAmqpIncomingPackage(msg, msg.MessageId, queue.Name())
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
