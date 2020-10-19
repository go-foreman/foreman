package amqp

import "github.com/go-foreman/foreman/pkg/pubsub/transport"

func Topic(name string, durable, autoDelete, internal, noWait bool) transport.Topic {
	return amqpTopic{topicName: name, durable: durable, autoDelete: autoDelete, internal: internal, noWait: noWait}
}

type amqpTopic struct {
	topicName  string
	durable    bool
	autoDelete bool
	internal   bool
	noWait     bool
}

func (a amqpTopic) Name() string {
	return a.topicName
}
