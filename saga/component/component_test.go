package component

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/go-foreman/foreman/pubsub/endpoint"

	endpointMock "github.com/go-foreman/foreman/testing/mocks/pubsub/endpoint"

	"github.com/go-foreman/foreman/saga/contracts"

	"github.com/go-foreman/foreman"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	sagaPkg "github.com/go-foreman/foreman/saga"
	"github.com/go-foreman/foreman/testing/log"
	messageMock "github.com/go-foreman/foreman/testing/mocks/pubsub/message"
	"github.com/go-foreman/foreman/testing/mocks/pubsub/subscriber"
	"github.com/go-foreman/foreman/testing/mocks/saga/mutex"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-foreman/foreman/testing/mocks/saga"
	"github.com/golang/mock/gomock"
)

func TestComponent_Init(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := log.NewNilLogger()
	msgMarshallerMock := messageMock.NewMockMarshaller(ctrl)
	schemeRegistry := scheme.NewKnownTypesRegistry()
	subscriberInstanceMock := subscriber.NewMockSubscriber(ctrl)

	mBus, err := foreman.NewMessageBus(testLogger, msgMarshallerMock, schemeRegistry, foreman.WithSubscriber(subscriberInstanceMock))
	require.NoError(t, err)

	storeMock := saga.NewMockStore(ctrl)
	mutexMock := mutex.NewMockMutex(ctrl)

	t.Run("store factory return an error", func(t *testing.T) {
		c := NewSagaComponent(
			func(msgMarshaller message.Marshaller) (sagaPkg.Store, error) {
				return storeMock, errors.New("some error")
			},
			mutexMock,
		)

		err := c.Init(mBus)
		assert.Error(t, err)
		assert.EqualError(t, err, "some error")
	})

	t.Run("init component with no errors", func(t *testing.T) {
		endpointInstanceMock := endpointMock.NewMockEndpoint(ctrl)
		c := NewSagaComponent(
			func(msgMarshaller message.Marshaller) (sagaPkg.Store, error) {
				return storeMock, nil
			},
			mutexMock,
		)

		group := scheme.Group("test")
		mBus.SchemeRegistry().AddKnownTypes(group, &dataContract{})

		sagaExample := &sagaExample{}

		c.RegisterSagas(sagaExample)
		c.RegisterSagaEndpoints(endpointInstanceMock)

		err := c.Init(mBus)
		assert.NoError(t, err)

		systemExec1 := mBus.Dispatcher().Match(&contracts.StartSagaCommand{})
		systemExec2 := mBus.Dispatcher().Match(&contracts.RecoverSagaCommand{})
		systemExec3 := mBus.Dispatcher().Match(&contracts.CompensateSagaCommand{})
		require.True(t, len(systemExec1) == len(systemExec2) && len(systemExec2) == len(systemExec3))
		require.Len(t, systemExec1, 1)

		funcName1 := runtime.FuncForPC(reflect.ValueOf(systemExec1[0]).Pointer()).Name()
		funcName2 := runtime.FuncForPC(reflect.ValueOf(systemExec2[0]).Pointer()).Name()
		funcName3 := runtime.FuncForPC(reflect.ValueOf(systemExec3[0]).Pointer()).Name()

		assert.Equal(t, funcName1, funcName2)
		assert.Equal(t, funcName2, funcName3)
		assert.Equal(t, funcName1, "github.com/go-foreman/foreman/saga/handlers.SagaControlHandler.Handle-fm")

		sagaHandlersRegistered := mBus.Dispatcher().Match(&dataContract{})
		require.Len(t, sagaHandlersRegistered, 1)
		sagaHandlerFuncName := runtime.FuncForPC(reflect.ValueOf(sagaHandlersRegistered[0]).Pointer()).Name()
		assert.Equal(t, sagaHandlerFuncName, "github.com/go-foreman/foreman/saga/handlers.SagaEventsHandler.Handle-fm")

		gk, err := mBus.SchemeRegistry().ObjectKind(&contracts.StartSagaCommand{})
		require.NoError(t, err)
		assert.Equal(t, gk, &scheme.GroupKind{Group: "systemSaga", Kind: "StartSagaCommand"})

		gk, err = mBus.SchemeRegistry().ObjectKind(&contracts.RecoverSagaCommand{})
		require.NoError(t, err)
		assert.Equal(t, gk, &scheme.GroupKind{Group: "systemSaga", Kind: "RecoverSagaCommand"})

		gk, err = mBus.SchemeRegistry().ObjectKind(&contracts.CompensateSagaCommand{})
		require.NoError(t, err)
		assert.Equal(t, gk, &scheme.GroupKind{Group: "systemSaga", Kind: "CompensateSagaCommand"})

		gk, err = mBus.SchemeRegistry().ObjectKind(&contracts.SagaCompletedEvent{})
		require.NoError(t, err)
		assert.Equal(t, gk, &scheme.GroupKind{Group: "systemSaga", Kind: "SagaCompletedEvent"})

		gk, err = mBus.SchemeRegistry().ObjectKind(&contracts.SagaChildCompletedEvent{})
		require.NoError(t, err)
		assert.Equal(t, gk, &scheme.GroupKind{Group: "systemSaga", Kind: "SagaChildCompletedEvent"})

		allContracts := []message.Object{
			&contracts.StartSagaCommand{},
			&contracts.RecoverSagaCommand{},
			&contracts.CompensateSagaCommand{},
			&contracts.SagaCompletedEvent{},
			&contracts.SagaChildCompletedEvent{},
			&dataContract{},
		}

		endpoints := mBus.Router().Route(allContracts[0])

		assert.Len(t, endpoints, 1)
		assert.Equal(t, endpoints, []endpoint.Endpoint{endpointInstanceMock})

		for _, contr := range allContracts {
			currentEndpoints := mBus.Router().Route(contr)
			assert.Equal(t, endpoints, mBus.Router().Route(contr))
			endpoints = currentEndpoints
		}
	})
}

type sagaExample struct {
	sagaPkg.BaseSaga
}

func (s *sagaExample) Init() {
	s.AddEventHandler(&dataContract{}, s.HandleData)
}

func (s *sagaExample) Start(sagaCtx sagaPkg.SagaContext) error {
	//TODO implement me
	panic("implement me")
}

func (s *sagaExample) Compensate(sagaCtx sagaPkg.SagaContext) error {
	//TODO implement me
	panic("implement me")
}

func (s *sagaExample) Recover(sagaCtx sagaPkg.SagaContext) error {
	//TODO implement me
	panic("implement me")
}

func (s *sagaExample) HandleData(sagaCtx sagaPkg.SagaContext) error {
	return nil
}

type dataContract struct {
	message.ObjectMeta
}
