package amqp

import "github.com/go-foreman/foreman/pubsub/transport"

func Queue(name string, durable, autoDelete, exclusive, noWait bool) transport.Queue {
	return amqpQueue{queueName: name, durable: durable, autoDelete: autoDelete, exclusive: exclusive, noWait: noWait}
}

type amqpQueue struct {
	queueName  string
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
