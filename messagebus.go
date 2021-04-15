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
type SubscriberOption func(subscriberOpts *subscriberOpts, c *subscriberContainer)

type subscriberOpts struct {
	subscriber subscriber.Subscriber
	transport  transport.Transport
}

type subscriberContainer struct {
	msgMarshaller message.Marshaller
	processor     subscriber.Processor
}

// WithSubscriber option allows to specify your own implementation of Subscriber for MessageBus
func WithSubscriber(subscriber subscriber.Subscriber) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *subscriberContainer) {
		subscriberOpts.subscriber = subscriber
	}
}

// DefaultWithTransport option allows to specify your own transport which will be used in the default subscriber
func DefaultWithTransport(transport transport.Transport) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *subscriberContainer) {
		subscriberOpts.transport = transport
	}
}

// SubscriberFactory is a function which gives you an access to default Processor and message.Decoder, these are needed when you implement own Subscriber
type SubscriberFactory func(processor subscriber.Processor, marshaller message.Marshaller) subscriber.Subscriber

// WithSubscriberFactory provides a way to construct your own Subscriber and pass it along to MessageBus
func WithSubscriberFactory(factory SubscriberFactory) SubscriberOption {
	return func(subscriberOpts *subscriberOpts, c *subscriberContainer) {
		subscriberOpts.subscriber = factory(c.processor, c.msgMarshaller)
	}
}

// ConfigOption allows to configure MessageBus's container
type ConfigOption func(o *container)

type container struct {
	messageExuctionCtxFactory execution.MessageExecutionCtxFactory
	messagesDispatcher        dispatcher.Dispatcher
	router                    endpoint.Router
	logger                    log.Logger
	msgMarshaller             message.Marshaller
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

func WithLogger(logger log.Logger) ConfigOption {
	return func(c *container) {
		c.logger = logger
	}
}

// MessageBus is a main component, kind of a container which aggregates other components
type MessageBus struct {
	marshaller         message.Marshaller
	messagesDispatcher dispatcher.Dispatcher
	router             endpoint.Router
	scheme             scheme.KnownTypesRegistry
	subscriber         subscriber.Subscriber
	logger             log.Logger
}

// NewMessageBus constructs MessageBus, allows to specify logger, choose subscriber or use default with transport and other options which configure implementations of other important parts
func NewMessageBus(logger log.Logger, msgMarshaller message.Marshaller, scheme scheme.KnownTypesRegistry, subscriberOption SubscriberOption, configOpts ...ConfigOption) (*MessageBus, error) {
	mBus := &MessageBus{logger: logger, marshaller: msgMarshaller, scheme: scheme}

	container := &container{
		msgMarshaller: msgMarshaller,
	}
	for _, config := range configOpts {
		config(container)
	}

	if container.messagesDispatcher == nil {
		container.messagesDispatcher = dispatcher.NewDispatcher()
	}

	if container.router == nil {
		container.router = endpoint.NewRouter()
	}

	if container.messageExuctionCtxFactory == nil {
		container.messageExuctionCtxFactory = execution.NewMessageExecutionCtxFactory(container.router, logger)
	}

	if container.processor == nil {
		container.processor = subscriber.NewMessageProcessor(msgMarshaller, container.messageExuctionCtxFactory, container.messagesDispatcher, logger)
	}

	mBus.messagesDispatcher = container.messagesDispatcher
	mBus.router = container.router
	mBus.scheme = container.scheme

	subscriberOpt := &subscriberOpts{}
	subscriberOption(subscriberOpt, &subscriberContainer{
		msgMarshaller: msgMarshaller,
		processor:     container.processor,
	})

	if subscriberOpt.subscriber != nil {
		mBus.subscriber = subscriberOpt.subscriber
	} else if subscriberOpt.transport != nil {
		mBus.subscriber = subscriber.NewSubscriber(subscriberOpt.transport, container.processor, logger)
	} else {
		return nil, errors.New("subscriber is nil")
	}

	for _, component := range container.components {
		if err := component.Init(mBus); err != nil {
			return nil, err
		}
	}

	return mBus, nil
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
