package transport

import (
	"context"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../../testing/mocks/pubsub/transport/transport.go -package transport . Transport

type Transport interface {
	// CreateTopic creates a topic(exchange) in message broker
	CreateTopic(ctx context.Context, topic Topic) error
	// CreateQueue creates a queue in a message broker
	CreateQueue(ctx context.Context, queue Queue, queueBind ...QueueBind) error
	// Consume starts receiving packages in a gorotuine and sends them to the <-chan IncomingPkg
	Consume(ctx context.Context, queues []Queue, options ...ConsumeOpts) (<-chan IncomingPkg, error)
	// Send sends an outbound package to a defined destination topic in OutboundPkg
	Send(ctx context.Context, outboundPkg OutboundPkg, options ...SendOpts) error
	// Connect connects to a message broker. It should be able to reconnect automatically in case of failure.
	Connect(context.Context) error
	// Disconnect disconnects from a message broker and stops listening for packages.
	Disconnect(context.Context) error
}

type Topic interface {
	Name() string
}

type Queue interface {
	Name() string
}

type QueueBind interface {
	DestinationTopic() string
	BindingKey() string
}

type ConsumeOpts func(options interface{}) error
type SendOpts func(options interface{}) error
