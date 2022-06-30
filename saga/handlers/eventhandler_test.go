package handlers

import (
	"context"
	"testing"
	"time"

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

	schemeRegistry.AddKnownTypes(g, &DataContract{})
	handler := NewEventsHandler(sagaStoreMock, sagaMutexMock, schemeRegistry, idService)

	t.Run("success", func(t *testing.T) {
		sagaID := "123"
		ev := &DataContract{Message: "something happened"}
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

		err := handler.Handle(msgExecutionCtx)
		assert.NoError(t, err)
	})
}
