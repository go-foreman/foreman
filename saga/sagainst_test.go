package saga

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/pubsub/message"
)

func TestInstance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sagaCtxMock := NewMockSagaContext(ctrl)

	sagaEx := &sagaExample{}
	instance := NewSagaInstance("123", "777", sagaEx)

	assert.Equal(t, instance.UID(), "123")
	assert.Equal(t, instance.ParentID(), "777")
	assert.Same(t, instance.Saga(), sagaEx)
	assert.Equal(t, instance.Status().String(), sagaStatusCreated.String())
	assert.Empty(t, instance.HistoryEvents())
	assert.Nil(t, instance.Status().FailedOnEvent())

	instance.AddHistoryEvent(&DataContract{}, &AddHistoryEvent{
		TraceUID: "xxx",
		Origin:   "origin",
	})

	require.Len(t, instance.HistoryEvents(), 1)

	historyEv := instance.HistoryEvents()[0]
	assert.Equal(t, historyEv.TraceUID, "xxx")
	assert.Equal(t, historyEv.OriginSource, "origin")
	assert.Equal(t, historyEv.Payload, &DataContract{})

	assert.Nil(t, instance.StartedAt())
	assert.Nil(t, instance.UpdatedAt())

	sagaCtxMock.EXPECT().Dispatch(&DataContract{Message: "start"})

	err := instance.Start(sagaCtxMock)
	assert.NoError(t, err)
	assert.Equal(t, instance.Status().String(), sagaStatusInProgress.String())
	assert.True(t, instance.Status().InProgress())

	sagaCtxMock.EXPECT().Dispatch(&DataContract{Message: "recover"})
	sagaCtxMock.EXPECT().SagaInstance().Return(instance)

	err = instance.Recover(sagaCtxMock)
	assert.NoError(t, err)
	assert.Equal(t, instance.Status().String(), sagaStatusRecovering.String())
	assert.True(t, instance.Status().Recovering())

	failedEv := &DataContract{Message: "failed here"}
	instance.Fail(failedEv)

	assert.Equal(t, instance.Status().FailedOnEvent(), failedEv)

	sagaCtxMock.EXPECT().Dispatch(failedEv)
	sagaCtxMock.EXPECT().SagaInstance().Return(instance)

	err = instance.Recover(sagaCtxMock)
	assert.NoError(t, err)

	sagaCtxMock.EXPECT().Dispatch(&DataContract{Message: "compensate"})
	err = instance.Compensate(sagaCtxMock)

	assert.NoError(t, err)
	assert.Equal(t, instance.Status().String(), sagaStatusCompensating.String())
	assert.True(t, instance.Status().Compensating())

	instance.Complete()
	assert.Equal(t, instance.Status().String(), sagaStatusCompleted.String())
	assert.True(t, instance.Status().Completed())

}

type sagaExample struct {
	BaseSaga
}

func (s *sagaExample) Init() {
	s.AddEventHandler(&DataContract{}, s.HandleData)
}

func (s *sagaExample) Start(sagaCtx SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "start"})
	return nil
}

func (s *sagaExample) Compensate(sagaCtx SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "compensate"})
	return nil
}

func (s *sagaExample) Recover(sagaCtx SagaContext) error {
	if failedEv := sagaCtx.SagaInstance().Status().FailedOnEvent(); failedEv != nil {
		sagaCtx.Dispatch(failedEv)
		return nil
	}

	sagaCtx.Dispatch(&DataContract{Message: "recover"})
	return nil
}

func (s *sagaExample) HandleData(sagaCtx SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "handle"})
	return nil
}

type DataContract struct {
	message.ObjectMeta
	Message string
}
