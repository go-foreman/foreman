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
	url        string
	connection *amqp.Connection
	ch         *amqp.Channel
	logger     log.Logger
}

func (t *amqpTransport) Connect(ctx context.Context) error {
	conn, err := amqp.Dial(t.url)
	if err != nil {
		return errors.WithStack(err)
	}

	ch, err := conn.Channel()

	if err != nil {
		return errors.WithStack(err)
	}

	t.connection = conn
	t.ch = ch

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

	err := t.ch.ExchangeDeclare(
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

	_, err := t.ch.QueueDeclare(
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
		err := t.ch.QueueBind(
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

	err := t.ch.Publish(
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

	income := make(chan pkg.IncomingPkg)

	consumersWait := sync.WaitGroup{}

	for _, q := range queues {
		consumersWait.Add(1)
		go func(queue transport.Queue) {
			defer consumersWait.Done()

			ch, err := t.connection.Channel()

			if err != nil {
				t.logger.Log(log.ErrorLevel, err)
				return
			}

			defer func() {
				ch.Close()
			}()

			msgs, err := ch.Consume(
				queue.Name(),
				consumeOptions.Consumer,
				consumeOptions.AutoAck,
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
					inPkg := pkg.NewAmqpIncomingPackage(msg, msg.MessageId, queue.Name())

					income <- inPkg
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
	if t.connection == nil || t.ch == nil {
		return nil
	}

	if err := t.ch.Close(); err != nil {
		return errors.Wrap(err, "error closing channel")
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
