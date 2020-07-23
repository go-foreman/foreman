package transport

import (
	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
)

type Transport interface {
	CreateTopic(ctx context.Context, topic Topic) error
	CreateQueue(ctx context.Context, queue Queue, queueBind ...QueueBind) error
	Consume(ctx context.Context, queues []Queue, options ...ConsumeOpts) (<-chan pkg.IncomingPkg, error)
	Send(ctx context.Context, outboundPkg pkg.OutboundPkg, options ...SendOpts) error
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
