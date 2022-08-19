package amqp

import "github.com/go-foreman/foreman/pubsub/transport"

type QueueType string

const (
	QueueTypeClassic QueueType = "classic"
	QueueTypeQuorum  QueueType = "quorum"
)

type QueueOptionsPatch func(options *amqpQueue)

func WithQueueType(v QueueType) QueueOptionsPatch {
	return func(options *amqpQueue) {
		options.queueType = v
	}
}

func Queue(name string, durable, autoDelete, exclusive, noWait bool, patches ...QueueOptionsPatch) transport.Queue {
	q := amqpQueue{queueName: name, durable: durable, autoDelete: autoDelete, exclusive: exclusive, noWait: noWait}

	for _, patch := range patches {
		patch(&q)
	}

	return q
}

type amqpQueue struct {
	queueName  string
	queueType  QueueType
	durable    bool
	autoDelete bool
	exclusive  bool
	noWait     bool
}

func (q amqpQueue) Name() string {
	return q.queueName
}

func QueueBind(destinationTopic, bindingKey string, noWait bool) transport.QueueBind {
	return amqpQueueBind{destination: destinationTopic, binding: bindingKey, noWait: noWait}
}

type amqpQueueBind struct {
	destination string
	binding     string
	noWait      bool
}

func (q amqpQueueBind) DestinationTopic() string {
	return q.destination
}

func (q amqpQueueBind) BindingKey() string {
	return q.binding
}
