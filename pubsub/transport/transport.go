package transport

import (
	"context"
)

type Transport interface {
	CreateTopic(ctx context.Context, topic Topic) error
	CreateQueue(ctx context.Context, queue Queue, queueBind ...QueueBind) error
	Consume(ctx context.Context, queues []Queue, options ...ConsumeOpts) (<-chan IncomingPkg, error)
	Send(ctx context.Context, outboundPkg OutboundPkg, options ...SendOpts) error
	Connect(context.Context) error
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
