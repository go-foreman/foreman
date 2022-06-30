package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/go-foreman/foreman/saga/contracts"

	"github.com/go-foreman/foreman/pubsub/endpoint"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/saga"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/testing/log"
	"github.com/go-foreman/foreman/testing/mocks/pubsub/message/execution"
	sagaMocks "github.com/go-foreman/foreman/testing/mocks/saga"
	"github.com/go-foreman/foreman/testing/mocks/saga/mutex"
	"github.com/golang/mock/gomock"
)

func TestEventHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sagaStoreMock := sagaMocks.NewMockStore(ctrl)
	sagaMutexMock := mutex.NewMockMutex(ctrl)
	idService := sagaMocks.NewMockSagaUIDService(ctrl)
	schemeRegistry := scheme.NewKnownTypesRegistry()
	testLogger := log.NewNilLogger()

	msgExecutionCtx := execution.NewMockMessageExecutionCtx(ctrl)
	ctx := context.Background()
	g := scheme.Group("example")

	sagaObj := &SagaExample{
		BaseSaga: saga.BaseSaga{
			ObjectMeta: message.ObjectMeta{
				TypeMeta: scheme.TypeMeta{
					Kind:  "SagaExample",
					Group: g.String(),
				},
			},
		},
		Data: "data",
		err:  nil,
	}
	evObjMeta := message.ObjectMeta{
		TypeMeta: scheme.TypeMeta{
			Kind:  "DataContract",
			Group: g.String(),
		},
	}

	schemeRegistry.AddKnownTypes(g, &DataContract{})
	handler := NewEventsHandler(sagaStoreMock, sagaMutexMock, schemeRegistry, idService)

	t.Run("success", func(t *testing.T) {
		sagaID := "123"
		ev := &DataContract{
			ObjectMeta: evObjMeta,
			Message:    "something happened",
		}
		receivedMsg := message.NewReceivedMessage(sagaID, ev, message.Headers{}, time.Now(), "origin")

		sagaInstance := saga.NewSagaInstance(sagaID, "777", sagaObj)

		msgExecutionCtx.EXPECT().Message().Return(receivedMsg)
		msgExecutionCtx.EXPECT().Context().Return(ctx)
		msgExecutionCtx.EXPECT().Logger().Return(testLogger).Times(2)

		idService.EXPECT().ExtractSagaUID(receivedMsg.Headers()).Return(sagaID, nil)

		lockMock := mutex.NewMockLock(ctrl)
		sagaMutexMock.EXPECT().Lock(ctx, sagaID).Return(lockMock, nil)
		lockMock.EXPECT().Release(gomock.Any()).Return(errors.New("error releasing mutex"))

		sagaStoreMock.EXPECT().GetById(ctx, sagaID).Return(sagaInstance, nil)
		sagaStoreMock.EXPECT().Update(ctx, sagaInstance).Return(nil)

		idService.EXPECT().AddSagaId(receivedMsg.Headers(), sagaID).Return()

		msgExecutionCtx.
			EXPECT().
			Send(gomock.Any()).
			DoAndReturn(func(msg *message.OutcomingMessage, options ...endpoint.DeliveryOption) error {
				assert.Equal(t, msg.Payload(), &DataContract{Message: "handle"})
				return nil
			})

		err := handler.Handle(msgExecutionCtx)
		assert.NoError(t, err)
	})

	t.Run("success with parent id", func(t *testing.T) {
		defer testLogger.Clear()

		sagaObj := &SagaExample{
			BaseSaga: sagaObj.BaseSaga,
			Data:     "data",
			handleCallback: func(sagaInst saga.Instance) {
				sagaInst.Complete()
			},
		}

		sagaID := "123"
		ev := &DataContract{
			ObjectMeta: evObjMeta,
			Message:    "something happened",
		}
		receivedMsg := message.NewReceivedMessage(sagaID, ev, message.Headers{}, time.Now(), "origin")

		sagaInstance := saga.NewSagaInstance(sagaID, "777", sagaObj)

		msgExecutionCtx.EXPECT().Message().Return(receivedMsg)
		msgExecutionCtx.EXPECT().Context().Return(ctx)
		msgExecutionCtx.EXPECT().Logger().Return(testLogger).Times(2)

		idService.EXPECT().ExtractSagaUID(receivedMsg.Headers()).Return(sagaID, nil)

		lockMock := mutex.NewMockLock(ctrl)
		sagaMutexMock.EXPECT().Lock(ctx, sagaID).Return(lockMock, nil)
		lockMock.EXPECT().Release(gomock.Any()).Return(errors.New("error releasing mutex"))

		sagaStoreMock.EXPECT().GetById(ctx, sagaID).Return(sagaInstance, nil)
		sagaStoreMock.EXPECT().Update(ctx, sagaInstance).Return(nil)

		idService.EXPECT().AddSagaId(receivedMsg.Headers(), sagaID).Return()

		msgExecutionCtx.
			EXPECT().
			Send(gomock.Any()).
			DoAndReturn(func(msg *message.OutcomingMessage, options ...endpoint.DeliveryOption) error {
				assert.Equal(t, msg.Payload(), &DataContract{Message: "handle"})
				return nil
			})

		idService.EXPECT().AddSagaId(receivedMsg.Headers(), "777").Return()

		msgExecutionCtx.
			EXPECT().
			Send(gomock.Any()).
			DoAndReturn(func(msg *message.OutcomingMessage, options ...endpoint.DeliveryOption) error {
				assert.Equal(t, msg.Payload(), &contracts.SagaChildCompletedEvent{SagaUID: sagaInstance.UID()})
				return nil
			})

		err := handler.Handle(msgExecutionCtx)
		assert.NoError(t, err)
	})

	t.Run("error extracting saga id", func(t *testing.T) {
		defer testLogger.Clear()

		ev := &DataContract{
			ObjectMeta: evObjMeta,
			Message:    "something happened",
		}
		receivedMsg := message.NewReceivedMessage("xxx", ev, message.Headers{}, time.Now(), "origin")

		msgExecutionCtx.EXPECT().Message().Return(receivedMsg)
		msgExecutionCtx.EXPECT().Context().Return(ctx)
		msgExecutionCtx.EXPECT().Logger().Return(testLogger)

		idService.EXPECT().ExtractSagaUID(receivedMsg.Headers()).Return("", errors.New("error parsing headers"))
		err := handler.Handle(msgExecutionCtx)
		assert.Error(t, err)
		assert.EqualError(t, err, "extracting saga id from message 'xxx': error parsing headers")
	})

	t.Run("error locking saga", func(t *testing.T) {
		defer testLogger.Clear()

		sagaID := "123"
		ev := &DataContract{
			ObjectMeta: evObjMeta,
			Message:    "something happened",
		}
		receivedMsg := message.NewReceivedMessage("xxx", ev, message.Headers{}, time.Now(), "origin")

		msgExecutionCtx.EXPECT().Message().Return(receivedMsg)
		msgExecutionCtx.EXPECT().Context().Return(ctx)
		msgExecutionCtx.EXPECT().Logger().Return(testLogger)

		idService.EXPECT().ExtractSagaUID(receivedMsg.Headers()).Return(sagaID, nil)
		sagaMutexMock.EXPECT().Lock(ctx, sagaID).Return(nil, errors.New("error locking mutex"))

		err := handler.Handle(msgExecutionCtx)
		assert.Error(t, err)
		assert.EqualError(t, err, "locking saga '123': error locking mutex")
	})

	t.Run("saga not found by id in store or returns an error", func(t *testing.T) {
		defer testLogger.Clear()

		times := 2
		sagaID := "123"
		ev := &DataContract{
			ObjectMeta: evObjMeta,
			Message:    "something happened",
		}
		receivedMsg := message.NewReceivedMessage(sagaID, ev, message.Headers{}, time.Now(), "origin")

		msgExecutionCtx.EXPECT().Message().Return(receivedMsg).Times(times)
		msgExecutionCtx.EXPECT().Context().Return(ctx).Times(times)
		msgExecutionCtx.EXPECT().Logger().Return(testLogger).Times(times)

		idService.EXPECT().ExtractSagaUID(receivedMsg.Headers()).Return(sagaID, nil).Times(times)

		lockMock := mutex.NewMockLock(ctrl)
		sagaMutexMock.EXPECT().Lock(ctx, sagaID).Return(lockMock, nil).Times(times)
		lockMock.EXPECT().Release(gomock.Any()).Return(errors.New("error releasing mutex")).Times(times)

		sagaStoreMock.EXPECT().GetById(ctx, sagaID).Return(nil, nil)

		err := handler.Handle(msgExecutionCtx)
		assert.Error(t, err)
		assert.EqualError(t, err, "saga '123' not found")

		sagaStoreMock.EXPECT().GetById(ctx, sagaID).Return(nil, errors.New("error getting by id"))

		err = handler.Handle(msgExecutionCtx)
		assert.Error(t, err)
		assert.EqualError(t, err, "retrieving saga '123' from store: error getting by id")
	})

}
