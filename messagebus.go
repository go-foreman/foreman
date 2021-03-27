package brigadier

import (
	"errors"
	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/dispatcher"
	"github.com/go-foreman/foreman/pubsub/endpoint"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/pubsub/subscriber"
	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/go-foreman/foreman/runtime/scheme"
)

// Component allow to wrap and prepare booting of your component, which will be initialized by MessageBus
type Component interface {
	Init(b *MessageBus) error
}

// SubscriberOption allows to provide a few options for configuring Subscriber
type SubscriberOption func(subscriberOpts *subscriberOpts, c *container)

type subscriberOpts struct {
	subscriber subscriber.Subscriber
	transport  transport.Transport
}

// WithSubscriber option allows to specify your own implementation of Subscriber for MessageBus
func WithSubscriber(subscriber subscriber.Subscriber) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *container) {
		subscriberOpts.subscriber = subscriber
	}
}

// DefaultWithTransport option allows to specify your own transport which will be used in the default subscriber
func DefaultWithTransport(transport transport.Transport) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *container) {
		subscriberOpts.transport = transport
	}
}

// SubscriberFactory is a function which gives you an access to default Processor and message.Decoder, these are needed when you implement own Subscriber
type SubscriberFactory func(processor subscriber.Processor, decoder message.Decoder) subscriber.Subscriber

// WithSubscriberFactory provides a way to construct your own Subscriber and pass it along to MessageBus
func WithSubscriberFactory(factory SubscriberFactory) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *container) {
		subscriberOpts.subscriber = factory(c.processor, c.messageDecoder)
	}
}

// ConfigOption allows to configure MessageBus's container
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

// WithComponents specifies a list of additional components you want to be registered in MessageBus
func WithComponents(components ...Component) ConfigOption {
	return func(c *container) {
		c.components = append(c.components, components...)
	}
}

// WithRouter allows to provide another endpoint.Router implementation
func WithRouter(router endpoint.Router) ConfigOption {
	return func(c *container) {
		c.router = router
	}
}

// WithDispatcher allows to provide another dispatcher.Dispatcher implementation
func WithDispatcher(dispatcher dispatcher.Dispatcher) ConfigOption {
	return func(c *container) {
		c.messagesDispatcher = dispatcher
	}
}

// WithMessageDecoder allows to provide another message.Decoder implementation
func WithMessageDecoder(decoder message.Decoder) ConfigOption {
	return func(c *container) {
		c.messageDecoder = decoder
	}
}

// WithSchemeRegistry allows to specify scheme scheme.KnownTypesRegistry
func WithSchemeRegistry(scheme scheme.KnownTypesRegistry) ConfigOption {
	return func(c *container) {
		c.scheme = scheme
	}
}

// WithMessageExecutionFactory allows to provide own execution.MessageExecutionCtxFactory
func WithMessageExecutionFactory(factory execution.MessageExecutionCtxFactory) ConfigOption {
	return func(c *container) {
		c.messageExuctionCtxFactory = factory
	}
}

// MessageBus is a main component, kind of a container which aggregates other components
type MessageBus struct {
	messagesDispatcher dispatcher.Dispatcher
	router             endpoint.Router
	scheme             scheme.KnownTypesRegistry
	subscriber         subscriber.Subscriber
	logger             log.Logger
}

// NewMessageBus constructs MessageBus, allows to specify logger, choose subscriber or use default with transport and other options which configure implementations of other important parts
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

// Dispatcher returns an instance of dispatcher.Dispatcher
func (b *MessageBus) Dispatcher() dispatcher.Dispatcher {
	return b.messagesDispatcher
}

// Router returns an instance of endpoint.Router
func (b *MessageBus) Router() endpoint.Router {
	return b.router
}

// SchemeRegistry returns an instance of current scheme.KnownTypesRegistry which should contain all the types of commands and events MB works with
func (b *MessageBus) SchemeRegistry() scheme.KnownTypesRegistry {
	return b.scheme
}

// Subscriber returns an instance of subscriber.Subscriber which controls the main flow of messages
func (b *MessageBus) Subscriber() subscriber.Subscriber {
	return b.subscriber
}

// Logger returns an instance of logger
func (b *MessageBus) Logger() log.Logger {
	return b.logger
}
