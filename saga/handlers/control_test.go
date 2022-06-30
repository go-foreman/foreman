package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/go-foreman/foreman/pubsub/endpoint"

	"github.com/go-foreman/foreman/testing/log"

	"github.com/go-foreman/foreman/saga/contracts"

	"github.com/go-foreman/foreman/pubsub/message"
	sagaPkg "github.com/go-foreman/foreman/saga"

	"github.com/go-foreman/foreman/testing/mocks/pubsub/message/execution"
	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/testing/mocks/saga"
	"github.com/go-foreman/foreman/testing/mocks/saga/mutex"
	"github.com/golang/mock/gomock"
)

func TestControlHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sagaStoreMock := saga.NewMockStore(ctrl)
	sagaMutexMock := mutex.NewMockMutex(ctrl)
	idService := saga.NewMockSagaUIDService(ctrl)
	schemeRegistry := scheme.NewKnownTypesRegistry()
	testLogger := log.NewNilLogger()

	msgExecutionCtx := execution.NewMockMessageExecutionCtx(ctrl)

	handler := NewSagaControlHandler(sagaStoreMock, sagaMutexMock, schemeRegistry, idService)

	t.Run("create saga", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			defer testLogger.Clear()

			startSagaCmd := &contracts.StartSagaCommand{
				ObjectMeta: message.ObjectMeta{
					TypeMeta: scheme.TypeMeta{
						Kind:  "StartSagaCommand",
						Group: "systemSaga",
					},
				},
				SagaUID:   "123",
				ParentUID: "",
				Saga: &sagaExample{
					Data: "data",
				},
			}

			now := time.Now()
			ctx := context.Background()

			receivedMsg := message.NewReceivedMessage("123", startSagaCmd, message.Headers{}, now, "origin")

			msgExecutionCtx.EXPECT().Message().Return(receivedMsg)
			msgExecutionCtx.EXPECT().Context().Return(ctx)
			msgExecutionCtx.EXPECT().Logger().Return(testLogger).Times(2)

			lockMock := mutex.NewMockLock(ctrl)
			sagaMutexMock.EXPECT().Lock(ctx, startSagaCmd.SagaUID).Return(lockMock, nil)
			lockMock.EXPECT().Release(ctx).Return(nil)

			var sagaInstance sagaPkg.Instance
			sagaStoreMock.
				EXPECT().
				Create(ctx, gomock.Any()).
				DoAndReturn(func(ctx context.Context, sagaInst sagaPkg.Instance) error {
					sagaInstance = sagaInst
					return nil
				})

			sagaStoreMock.
				EXPECT().
				Update(ctx, gomock.Any()).
				DoAndReturn(func(ctx context.Context, sagaInst sagaPkg.Instance) error {
					assert.Same(t, sagaInstance, sagaInst)
					return nil
				})

			idService.EXPECT().AddSagaId(receivedMsg.Headers(), startSagaCmd.SagaUID).Return()
			msgExecutionCtx.
				EXPECT().
				Send(gomock.Any()).
				DoAndReturn(func(msg *message.OutcomingMessage, options ...endpoint.DeliveryOption) error {
					assert.Equal(t, &DataContract{Message: "start"}, msg.Payload())
					return nil
				})

			err := handler.Handle(msgExecutionCtx)
			assert.NoError(t, err)

			assert.Len(t, sagaInstance.HistoryEvents(), 2)
		})
	})

}

type sagaExample struct {
	sagaPkg.BaseSaga
	Data string
}

func (s *sagaExample) Init() {
	s.AddEventHandler(&DataContract{}, s.HandleData)
}

func (s *sagaExample) Start(sagaCtx sagaPkg.SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "start"})
	return nil
}

func (s *sagaExample) Compensate(sagaCtx sagaPkg.SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "compensate"})
	return nil
}

func (s *sagaExample) Recover(sagaCtx sagaPkg.SagaContext) error {
	if failedEv := sagaCtx.SagaInstance().Status().FailedOnEvent(); failedEv != nil {
		sagaCtx.Dispatch(failedEv)
		return nil
	}

	sagaCtx.Dispatch(&DataContract{Message: "recover"})
	return nil
}

func (s *sagaExample) HandleData(sagaCtx sagaPkg.SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "handle"})
	return nil
}

type DataContract struct {
	message.ObjectMeta
	Message string
}
