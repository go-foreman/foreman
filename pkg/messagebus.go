package pkg

import (
	"errors"
	"github.com/kopaygorodsky/brigadier/pkg/log"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/dispatcher"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message/execution"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/subscriber"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
)

type Component interface {
	Init(b *MessageBus) error
}

type SubscriberOption func(subscriberOpts *subscriberOpts, c *container)

type subscriberOpts struct {
	subscriber subscriber.Subscriber
	transport  transport.Transport
}

func WithSubscriber(subscriber subscriber.Subscriber) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *container) {
		subscriberOpts.subscriber = subscriber
	}
}

func DefaultWithTransport(transport transport.Transport) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *container) {
		subscriberOpts.transport = transport
	}
}

type SubscriberFactory func(processor subscriber.Processor, decoder message.Decoder) subscriber.Subscriber

func WithSubscriberFactory(factory SubscriberFactory) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *container) {
		subscriberOpts.subscriber = factory(c.processor, c.messageDecoder)
	}
}

type ConfigOption func(o *container)

type container struct {
	messageExuctionCtxFactory execution.MessageExecutionCtxFactory
	messagesDispatcher        dispatcher.Dispatcher
	router                    endpoint.Router
	logger                    log.Logger
	messageDecoder            message.Decoder
	processor                 subscriber.Processor
	scheme                    scheme.KnownTypesRegistry
	components                []Component
}

func WithComponents(components ...Component) ConfigOption {
	return func(c *container) {
		c.components = append(c.components, components...)
	}
}

func WithRouter(router endpoint.Router) ConfigOption {
	return func(c *container) {
		c.router = router
	}
}

func WithDispatcher(dispatcher dispatcher.Dispatcher) ConfigOption {
	return func(c *container) {
		c.messagesDispatcher = dispatcher
	}
}

func WithMessageDecoder(decoder message.Decoder) ConfigOption {
	return func(c *container) {
		c.messageDecoder = decoder
	}
}

func WithSchemeRegistry(scheme scheme.KnownTypesRegistry) ConfigOption {
	return func(c *container) {
		c.scheme = scheme
	}
}

func WithMessageExecutionFactory(factory execution.MessageExecutionCtxFactory) ConfigOption {
	return func(c *container) {
		c.messageExuctionCtxFactory = factory
	}
}

type MessageBus struct {
	messagesDispatcher dispatcher.Dispatcher
	router             endpoint.Router
	scheme             scheme.KnownTypesRegistry
	subscriber         subscriber.Subscriber
	logger             log.Logger
}

func NewMessageBus(logger log.Logger, subscriberOption SubscriberOption, configOpts ...ConfigOption) (*MessageBus, error) {
	b := &MessageBus{logger: logger}

	opts := &container{}
	for _, config := range configOpts {
		config(opts)
	}

	if opts.scheme == nil {
		opts.scheme = scheme.NewKnownTypesRegistry()
	}

	if opts.messagesDispatcher == nil {
		opts.messagesDispatcher = dispatcher.NewDispatcher()
	}

	if opts.router == nil {
		opts.router = endpoint.NewRouter()
	}

	if opts.messageExuctionCtxFactory == nil {
		opts.messageExuctionCtxFactory = execution.NewMessageExecutionCtxFactory(opts.router, logger)
	}

	if opts.messageDecoder == nil {
		opts.messageDecoder = message.NewJsonDecoder(opts.scheme)
	}

	if opts.processor == nil {
		opts.processor = subscriber.NewMessageProcessor(opts.messageDecoder, opts.messageExuctionCtxFactory, opts.messagesDispatcher, logger)
	}

	b.messagesDispatcher = opts.messagesDispatcher
	b.router = opts.router
	b.scheme = opts.scheme

	subscriberOpt := &subscriberOpts{}
	subscriberOption(subscriberOpt, opts)

	if subscriberOpt.subscriber != nil {
		b.subscriber = subscriberOpt.subscriber
	} else if subscriberOpt.transport != nil {
		b.subscriber = subscriber.NewSubscriber(subscriberOpt.transport, opts.processor, logger)
	} else {
		return nil, errors.New("subscriber is nil")
	}

	for _, component := range opts.components {
		if err := component.Init(b); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (b *MessageBus) Dispatcher() dispatcher.Dispatcher {
	return b.messagesDispatcher
}

func (b *MessageBus) Router() endpoint.Router {
	return b.router
}

func (b *MessageBus) SchemeRegistry() scheme.KnownTypesRegistry {
	return b.scheme
}

func (b *MessageBus) Subscriber() subscriber.Subscriber {
	return b.subscriber
}

func (b *MessageBus) Logger() log.Logger {
	return b.logger
}