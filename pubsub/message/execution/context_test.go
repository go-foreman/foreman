package execution

import (
	"context"
	"testing"
	"time"

	mockLogger "github.com/go-foreman/foreman/testing/mocks/log"

	"github.com/pkg/errors"

	"github.com/go-foreman/foreman/pubsub/endpoint"

	"github.com/go-foreman/foreman/log"
	"github.com/stretchr/testify/require"

	testingLog "github.com/go-foreman/foreman/testing/log"

	"github.com/stretchr/testify/assert"

	endpointMock "github.com/go-foreman/foreman/testing/mocks/pubsub/endpoint"

	"github.com/go-foreman/foreman/pubsub/message"

	"github.com/golang/mock/gomock"
)

type someTestType struct {
	message.ObjectMeta
	Data string `json:"data"`
}

func TestMessageExecutionCtx_Send(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := testingLog.NewNilLogger()
	testEndpoint := endpointMock.NewMockEndpoint(ctrl)
	testRouter := endpointMock.NewMockRouter(ctrl)

	factory := NewMessageExecutionCtxFactory(testRouter, testLogger)

	t.Run("no endpoints defined", func(t *testing.T) {
		defer testLogger.Clear()

		receivedMessage := message.NewReceivedMessage("123", &someTestType{}, message.Headers{}, time.Now(), "bus")
		outcomingMsg := message.NewOutcomingMessage(&someTestType{})
		ctx := context.Background()

		testRouter.
			EXPECT().
			Route(outcomingMsg.Payload()).
			Return(nil).
			Times(1)

		execCtx := factory.CreateCtx(ctx, receivedMessage)
		err := execCtx.Send(outcomingMsg)
		assert.NoError(t, err)

		require.Len(t, testLogger.Entries(), 1)
		logEntry := testLogger.Entries()[0]

		assert.Contains(t, logEntry.Msg, "no endpoints defined for message")
		assert.Equal(t, logEntry.Level, log.WarnLevel)
	})

	t.Run("successfully sent", func(t *testing.T) {
		defer testLogger.Clear()

		ctx := context.Background()
		receivedMessage := message.NewReceivedMessage("123", &someTestType{}, message.Headers{}, time.Now(), "bus")
		outcomingMsg := message.NewOutcomingMessage(&someTestType{})

		sendingOpt := endpoint.WithDelay(time.Second)

		testEndpoint.
			EXPECT().
			Send(ctx, outcomingMsg, gomock.AssignableToTypeOf(sendingOpt)).
			Return(nil).
			Times(1)

		testRouter.
			EXPECT().
			Route(outcomingMsg.Payload()).
			Return([]endpoint.Endpoint{testEndpoint}).
			Times(1)

		execCtx := factory.CreateCtx(ctx, receivedMessage)
		err := execCtx.Send(outcomingMsg, sendingOpt)
		assert.NoError(t, err)
		assert.Empty(t, testLogger.Messages())
	})

	t.Run("error sending a message", func(t *testing.T) {
		defer testLogger.Clear()

		ctx := context.Background()
		receivedMessage := message.NewReceivedMessage("123", &someTestType{}, message.Headers{"traceId": "111"}, time.Now(), "bus")
		outcomingMsg := message.NewOutcomingMessage(&someTestType{})

		testEndpoint.
			EXPECT().
			Send(ctx, outcomingMsg).
			Return(errors.New("some error")).
			Times(1)

		testRouter.
			EXPECT().
			Route(outcomingMsg.Payload()).
			Return([]endpoint.Endpoint{testEndpoint}).
			Times(1)

		execCtx := factory.CreateCtx(ctx, receivedMessage)
		err := execCtx.Send(outcomingMsg)
		assert.Error(t, err)
		assert.EqualError(t, err, "some error")

		require.Len(t, testLogger.Entries(), 1)
		logEntry := testLogger.Entries()[0]

		assert.Contains(t, logEntry.Msg, "some error")
		assert.Equal(t, logEntry.Level, log.ErrorLevel)
	})
}

func TestMessageExecutionCtx_Return(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := testingLog.NewNilLogger()
	testEndpoint := endpointMock.NewMockEndpoint(ctrl)
	testRouter := endpointMock.NewMockRouter(ctrl)

	factory := NewMessageExecutionCtxFactory(testRouter, testLogger)

	testRouter.
		EXPECT().
		Route(&someTestType{}).
		Return([]endpoint.Endpoint{testEndpoint}).
		AnyTimes()

	t.Run("successfully returning a message", func(t *testing.T) {
		defer testLogger.Clear()

		ctx := context.Background()
		sendingOpt := endpoint.WithDelay(time.Second)

		receivedMessage := message.NewReceivedMessage("123", &someTestType{}, message.Headers{}, time.Now(), "bus")

		outcomingMsg := message.FromReceivedMsg(receivedMessage)
		outcomingMsg.Headers().RegisterReturn()

		testEndpoint.
			EXPECT().
			Send(ctx, outcomingMsg, gomock.AssignableToTypeOf(sendingOpt)).
			Return(nil).
			Times(1)

		execCtx := factory.CreateCtx(ctx, receivedMessage)
		err := execCtx.Return(sendingOpt)
		assert.NoError(t, err)

		assert.Equal(t, 1, outcomingMsg.Headers().ReturnsCount())
	})

	t.Run("error returning a message", func(t *testing.T) {
		defer testLogger.Clear()

		ctx := context.Background()
		sendingOpt := endpoint.WithDelay(time.Second)

		receivedMessage := message.NewReceivedMessage("123", &someTestType{}, message.Headers{}, time.Now(), "bus")

		outcomingMsg := message.FromReceivedMsg(receivedMessage)
		outcomingMsg.Headers().RegisterReturn()

		testEndpoint.
			EXPECT().
			Send(ctx, outcomingMsg, gomock.AssignableToTypeOf(sendingOpt)).
			Return(errors.New("some error")).
			Times(1)

		execCtx := factory.CreateCtx(ctx, receivedMessage)
		err := execCtx.Return(sendingOpt)
		assert.Error(t, err)
		assert.EqualError(t, err, "returning message 123: some error")

		//make sure that headers of original message weren't modified
		assert.Equal(t, 0, receivedMessage.Headers().ReturnsCount())
		assert.Contains(t, testLogger.LastMessage(), "error when returning a message")

	})
}

func TestMessageExecutionCtx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := mockLogger.NewMockLogger(ctrl)
	testRouter := endpointMock.NewMockRouter(ctrl)

	factory := NewMessageExecutionCtxFactory(testRouter, testLogger)

	receivedMessage := message.NewReceivedMessage("123", &someTestType{}, message.Headers{"traceId": "111"}, time.Now(), "bus")
	ctx := context.Background()

	fields := []log.Field{
		{
			Name: "uid",
			Val:  receivedMessage.UID(),
		},
		{
			Name: "traceId",
			Val:  "111",
		},
	}

	testLogger.
		EXPECT().
		WithFields(fields).
		Return(testLogger)

	execCtx := factory.CreateCtx(ctx, receivedMessage)

	assert.Equal(t, execCtx.Context(), ctx)
	assert.Same(t, execCtx.Logger(), testLogger)
	assert.True(t, execCtx.Valid())
	assert.Same(t, execCtx.Message(), receivedMessage)

}
