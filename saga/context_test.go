package saga

import (
	"context"
	"testing"
	"time"

	"github.com/go-foreman/foreman/pubsub/message"

	"github.com/go-foreman/foreman/pubsub/endpoint"

	"github.com/go-foreman/foreman/log"
	logMock "github.com/go-foreman/foreman/testing/mocks/log"

	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/testing/mocks/pubsub/message/execution"
	"github.com/golang/mock/gomock"
)

func TestSagaContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgExecCtxMock := execution.NewMockMessageExecutionCtx(ctrl)
	sagaInstance := NewSagaInstance("123", "123", &sagaExample{})

	loggerMock := logMock.NewMockLogger(ctrl)
	msgExecCtxMock.EXPECT().Logger().Return(loggerMock).AnyTimes()

	loggerMock.EXPECT().WithFields([]log.Field{{Name: sagaUIDKey, Val: sagaInstance.UID()}}).Return(loggerMock)

	runningCtx := context.Background()
	sagaCtx := NewSagaCtx(msgExecCtxMock, sagaInstance)

	msgExecCtxMock.EXPECT().Context().Return(runningCtx)

	ctx := sagaCtx.Context()
	assert.Same(t, ctx, runningCtx)
	assert.Same(t, loggerMock, sagaCtx.Logger())

	msgExecCtxMock.EXPECT().Valid().Return(true)
	assert.Equal(t, sagaCtx.Valid(), true)

	assert.Same(t, sagaCtx.SagaInstance(), sagaInstance)
	assert.Empty(t, sagaCtx.Deliveries())

	sagaCtx.Dispatch(&DataContract{}, endpoint.WithDelay(time.Second))
	assert.Len(t, sagaCtx.Deliveries(), 1)
	assert.Equal(t, sagaCtx.Deliveries()[0].Payload, &DataContract{})
	assert.Len(t, sagaCtx.Deliveries()[0].Options, 1)

	receivedMsg := message.NewReceivedMessage("123", &DataContract{}, message.Headers{}, time.Now(), "origin")
	msgExecCtxMock.EXPECT().Message().Return(receivedMsg)

	assert.Equal(t, sagaCtx.Message(), receivedMsg)

	loggerMock.EXPECT().Log(log.InfoLevel, "returning saga event").Return()
	msgExecCtxMock.EXPECT().Return().Return(nil)
	err := sagaCtx.Return()
	assert.NoError(t, err)
}
