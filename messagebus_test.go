package foreman

import (
	"testing"

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
	subscriberMock := subscriberMock.NewMockSubscriber(ctrl)

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

		mBus, err := NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, WithSubscriber(subscriberMock), append(opts, WithComponents(componentMock, erroredComponentMock))...)
		require.Error(t, err)
		assert.Nil(t, mBus)
		assert.EqualError(t, err, "component error")

		mBus, err = NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, WithSubscriber(subscriberMock), append(opts, WithComponents(componentMock))...)
		require.NoError(t, err)
		require.NotNil(t, mBus)

		assert.Same(t, mBus.Logger(), testLogger)
		assert.Same(t, mBus.Dispatcher(), dispatcherMock)
		assert.Same(t, mBus.Router(), routerMock)
		assert.Same(t, mBus.SchemeRegistry(), schemeRegistry)
		assert.Same(t, mBus.Marshaller(), msgMarshallerMock)
	})

	t.Run("create mbus without opts", func(t *testing.T) {
		mBus, err := NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, WithSubscriber(subscriberMock))
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
				return subscriberMock
			}),
		)
		require.NoError(t, err)
		assert.Same(t, subscriberMock, mBus.Subscriber())
	})
}
