package execution

import (
	"context"
	"testing"
	"time"

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

		testRouter.
			EXPECT().
			Route(outcomingMsg.Payload()).
			Return(nil).
			Times(1)

		execCtx := factory.CreateCtx(context.Background(), receivedMessage)
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
			Send(ctx, outcomingMsg, gomock.Any()).
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
		receivedMessage := message.NewReceivedMessage("123", &someTestType{}, message.Headers{}, time.Now(), "bus")
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

//func TestMessageExecutionCtx_Return(t *testing.T) {
//	ctrl := gomock.NewController(t)
//	defer ctrl.Finish()
//
//	testLogger := testingLog.NewNilLogger()
//	testEndpoint := endpointMock.NewMockEndpoint(ctrl)
//	testRouter := endpointMock.NewMockRouter(ctrl)
//
//	factory := NewMessageExecutionCtxFactory(testRouter, testLogger)
//
//}
