package transport

import (
	"context"
	"fmt"
	"strings"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../../testing/mocks/pubsub/transport/transport.go -package transport . Transport

type Transport interface {
	// CreateTopic creates a topic(exchange) in message broker
	CreateTopic(ctx context.Context, topic Topic) error
	// CreateQueue creates a queue in a message broker
	CreateQueue(ctx context.Context, queue Queue, queueBind ...QueueBind) error
	// Consume starts receiving packages in a goroutine and sends them to the <-chan IncomingPkg
	Consume(ctx context.Context, groups ...ConsumableQueueGroup) (<-chan IncomingPkg, error)
	// Send sends an outbound package to a defined destination topic in OutboundPkg
	Send(ctx context.Context, outboundPkg OutboundPkg, options ...SendOpt) error
	// Disconnect disconnects from publishing and consuming channels
	Disconnect(context.Context) error
}

type Topic interface {
	Name() string
}

type Queue interface {
	Name() string
}

type ConsumableQueueGroup interface {
	String() string
	Queues() []Queue
	Opts() []ConsumeOpt
}

type consumableQueueGroup struct {
	queues []Queue
	opts   []ConsumeOpt
}

func NewConsumableQueueGroup(queues []Queue, opts ...ConsumeOpt) ConsumableQueueGroup {
	return &consumableQueueGroup{queues: queues, opts: opts}
}

func (g *consumableQueueGroup) String() string {
	names := make([]string, len(g.queues))
	for i, q := range g.queues {
		names[i] = q.Name()
	}
	return fmt.Sprintf("[%s]", strings.Join(names, ","))
}

func (g *consumableQueueGroup) Queues() []Queue    { return g.queues }
func (g *consumableQueueGroup) Opts() []ConsumeOpt { return g.opts }

type QueueBind interface {
	DestinationTopic() string
	BindingKey() string
}

type ConsumeOpt func(options interface{}) error
type SendOpt func(options interface{}) error
