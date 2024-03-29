package foreman

import (
	"testing"

	"github.com/go-foreman/foreman/testing/mocks/pubsub/transport"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/subscriber"

	"github.com/pkg/errors"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/testing/log"
	messageMock "github.com/go-foreman/foreman/testing/mocks/pubsub/message"
	subscriberMock "github.com/go-foreman/foreman/testing/mocks/pubsub/subscriber"
	"github.com/stretchr/testify/require"

	"github.com/go-foreman/foreman/testing/mocks/pubsub/dispatcher"
	"github.com/go-foreman/foreman/testing/mocks/pubsub/endpoint"
	"github.com/go-foreman/foreman/testing/mocks/pubsub/message/execution"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMessageBusConfigOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dispatcherMock := dispatcher.NewMockDispatcher(ctrl)
	msgExecFactoryMock := execution.NewMockMessageExecutionCtxFactory(ctrl)
	routerMock := endpoint.NewMockRouter(ctrl)
	componentMock := &aComponent{}

	c := &container{}

	opts := []ConfigOption{
		WithDispatcher(dispatcherMock),
		WithMessageExecutionFactory(msgExecFactoryMock),
		WithRouter(routerMock),
		WithComponents(componentMock),
	}

	for _, o := range opts {
		o(c)
	}

	assert.Same(t, c.messagesDispatcher, dispatcherMock)
	assert.Same(t, c.router, routerMock)
	assert.Equal(t, []Component{componentMock}, c.components)
	assert.Same(t, c.messageExuctionCtxFactory, msgExecFactoryMock)
}

type aComponent struct {
	err error
}

func (a aComponent) Init(b *MessageBus) error {
	return a.err
}

func TestMessageBusConstructor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := log.NewNilLogger()
	msgMarshallerMock := messageMock.NewMockMarshaller(ctrl)
	schemeRegistry := scheme.NewKnownTypesRegistry()
	subscriberInstanceMock := subscriberMock.NewMockSubscriber(ctrl)

	dispatcherMock := dispatcher.NewMockDispatcher(ctrl)
	msgExecFactoryMock := execution.NewMockMessageExecutionCtxFactory(ctrl)
	routerMock := endpoint.NewMockRouter(ctrl)
	componentMock := &aComponent{}
	erroredComponentMock := &aComponent{err: errors.New("component error")}

	t.Run("create mbus with all opts", func(t *testing.T) {
		opts := []ConfigOption{
			WithDispatcher(dispatcherMock),
			WithMessageExecutionFactory(msgExecFactoryMock),
			WithRouter(routerMock),
		}

		mBus, err := NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, WithSubscriber(subscriberInstanceMock), append(opts, WithComponents(componentMock, erroredComponentMock))...)
		require.Error(t, err)
		assert.Nil(t, mBus)
		assert.EqualError(t, err, "component error")

		mBus, err = NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, WithSubscriber(subscriberInstanceMock), append(opts, WithComponents(componentMock))...)
		require.NoError(t, err)
		require.NotNil(t, mBus)

		assert.Same(t, mBus.Logger(), testLogger)
		assert.Same(t, mBus.Dispatcher(), dispatcherMock)
		assert.Same(t, mBus.Router(), routerMock)
		assert.Same(t, mBus.SchemeRegistry(), schemeRegistry)
		assert.Same(t, mBus.Marshaller(), msgMarshallerMock)
	})

	t.Run("create mbus without opts", func(t *testing.T) {
		mBus, err := NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, WithSubscriber(subscriberInstanceMock))
		require.NoError(t, err)
		require.NotNil(t, mBus)

		assert.Same(t, mBus.scheme, mBus.SchemeRegistry())
		assert.Same(t, mBus.marshaller, mBus.Marshaller())
		assert.Same(t, mBus.messagesDispatcher, mBus.Dispatcher())
		assert.Same(t, mBus.router, mBus.Router())
		assert.Same(t, mBus.logger, mBus.Logger())
		assert.Same(t, mBus.subscriber, mBus.Subscriber())
	})

	t.Run("create mbus with custom subscriber", func(t *testing.T) {
		mBus, err := NewMessageBus(
			testLogger,
			msgMarshallerMock,
			schemeRegistry,
			WithSubscriberFactory(func(processor subscriber.Processor, marshaller message.Marshaller) subscriber.Subscriber {
				return subscriberInstanceMock
			}),
		)
		require.NoError(t, err)
		assert.Same(t, subscriberInstanceMock, mBus.Subscriber())
	})

	t.Run("create mbus with default subscriber", func(t *testing.T) {
		transportMock := transport.NewMockTransport(ctrl)

		sConfig := &subscriber.Config{}
		opts := []subscriber.Opt{subscriber.WithConfig(sConfig)}
		subscriberConfig := DefaultSubscriber(transportMock, opts...)
		sOpts := &subscriberOpts{}
		sContainer := &subscriberContainer{}
		subscriberConfig(sOpts, sContainer)

		assert.Same(t, sOpts.transport, transportMock)
		assert.Equal(t, sOpts.opts, opts)

		mBus, err := NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, DefaultSubscriber(transportMock))
		require.NoError(t, err)
		defaultSubscriber := subscriber.NewSubscriber(transportMock, nil, nil)
		assert.IsType(t, defaultSubscriber, mBus.Subscriber())
	})

	t.Run("nil subscriber", func(t *testing.T) {
		assert.PanicsWithError(t, "subscriber is nil", func() {
			_, _ = NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, WithSubscriber(nil))
		})
	})
}
