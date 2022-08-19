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

func NewTransport(conn UnderlyingConnection, logger log.Logger) transport.Transport {
	return &amqpTransport{
		connection: &Connection{
			underlyingConn:      conn,
			logger:              logger,
			chReconnectionDelay: delay,
		},
		mutex:             &sync.Mutex{},
		consumingChannels: map[AmqpChannel]struct{}{},
		logger:            logger,
	}
}

type amqpTransport struct {
	connection        AmqpConnection
	publishingChannel AmqpChannel
	mutex             *sync.Mutex
	consumingChannels map[AmqpChannel]struct{}
	logger            log.Logger
}

// CreateTopic creates an exchange in amqp. Allows options are: durable, autoDelete, internal, noWait.
func (t *amqpTransport) CreateTopic(ctx context.Context, topic transport.Topic) error {
	if err := t.checkConnection(); err != nil {
		return errors.WithStack(err)
	}

	amqpTopic, topicConv := topic.(amqpTopic)

	if !topicConv {
		return errors.Errorf("supplied topic is not an instance of amqp.Topic")
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
		return errors.Errorf("supplied Queue is not an instance of amqp.amqpQueue")
	}

	var queueBinds []amqpQueueBind

	for _, item := range qbs {
		queueBind, queueBindConv := item.(amqpQueueBind)

		if !queueBindConv {
			return errors.Errorf("one of supplied QueueBinds is not an instance of amqp.amqpQueueBind")
		}

		queueBinds = append(queueBinds, queueBind)
	}

	var table amqp.Table

	switch queue.queueType {
	case QueueTypeQuorum:
		table = amqp.Table{
			"x-queue-type": "quorum",
		}
	case QueueTypeClassic:
		table = amqp.Table{
			"x-queue-type": "classic",
		}
	}

	if _, err := t.publishingChannel.QueueDeclare(
		queue.Name(),
		queue.durable,
		queue.autoDelete,
		queue.exclusive,
		queue.noWait,
		table,
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

func (t *amqpTransport) Send(ctx context.Context, outboundPkg transport.OutboundPkg, options ...transport.SendOpt) error {
	if err := t.checkConnection(); err != nil {
		return errors.WithStack(err)
	}

	sendOptions := &sendOptions{}

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

func (t *amqpTransport) Consume(ctx context.Context, queues []transport.Queue, options ...transport.ConsumeOpt) (<-chan transport.IncomingPkg, error) {
	if err := t.checkConnection(); err != nil {
		return nil, errors.WithStack(err)
	}

	consumingChannel, err := t.connection.Channel()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	t.mutex.Lock()
	t.consumingChannels[consumingChannel] = struct{}{}
	t.mutex.Unlock()

	consumeOptions := &consumeOptions{}

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

	var consumersErr error

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
			consumersErr = errors.Wrapf(err, "consuming %s", q.Name())
			break
		}

		consumersWait.Add(1)

		go func(consumersCtx context.Context, queue transport.Queue, deliveries <-chan amqp.Delivery) {
			defer consumersWait.Done()

			defer func() {
				t.logger.Logf(log.InfoLevel, "canceling consumer %s", queue.Name())
				if err := consumingChannel.Cancel(queue.Name(), false); err != nil {
					t.logger.Logf(log.ErrorLevel, "error canceling consumer %s. %s", queue.Name(), err)
				} else {
					t.logger.Logf(log.InfoLevel, "canceled consumer %s", queue.Name())
				}
			}()

			for {
				select {
				case msg, open := <-deliveries:
					if !open {
						t.logger.Logf(log.WarnLevel, "amqp consumer closed channel for queue %s", queue.Name())
						return
					}

					select {
					case income <- &inAmqpPkg{origin: queue.Name(), receivedAt: time.Now(), delivery: &delivery{msg: &msg}}:
					case <-consumersCtx.Done():
						break
					}
				case <-consumersCtx.Done():
					t.logger.Logf(log.WarnLevel, "canceled context. Stopped consuming queue %s", queue.Name())
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

		t.mutex.Lock()
		delete(t.consumingChannels, consumingChannel)
		t.mutex.Unlock()
	}()

	if consumersErr != nil {
		cancelConsumers()
		return nil, consumersErr
	}

	return income, nil //nolint:govet
}

func (t *amqpTransport) Disconnect(ctx context.Context) error {

	if t.connection == nil || t.publishingChannel == nil {
		return nil
	}

	if err := t.publishingChannel.Close(); err != nil {
		return errors.Wrap(err, "closing publishing channel")
	}

	t.mutex.Lock()

	for ch := range t.consumingChannels {
		if err := ch.Close(); err != nil {
			return errors.Wrap(err, "closing one of consuming channels")
		}
	}

	t.mutex.Unlock()

	return nil
}

func (t *amqpTransport) checkConnection() error {
	if t.connection == nil {
		return errors.Errorf("connection is nil")
	}

	if t.publishingChannel == nil {
		ch, err := t.connection.Channel()

		if err != nil {
			return errors.Wrap(err, "creating publishing channel")
		}

		t.publishingChannel = ch
	}

	return nil
}
