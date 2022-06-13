package subscriber

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pkg/errors"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	mockTransport "github.com/go-foreman/foreman/testing/mocks/pubsub/transport"

	"github.com/go-foreman/foreman/testing/log"
	mockDispatcher "github.com/go-foreman/foreman/testing/mocks/pubsub/dispatcher"

	mockMessage "github.com/go-foreman/foreman/testing/mocks/pubsub/message"
	"github.com/golang/mock/gomock"
)

type someTest struct {
	message.ObjectMeta
	Data string `json:"data"`
}

func TestProcessor(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := log.NewNilLogger()
	marshaller := mockMessage.NewMockMarshaller(ctrl)
	dispatcher := mockDispatcher.NewMockDispatcher(ctrl)

	execCtxFactory := execution.NewMessageExecutionCtxFactory(nil, testLogger)

	pkgProcessor := NewMessageProcessor(marshaller, execCtxFactory, dispatcher, testLogger)

	data := &someTest{
		Data: "111",
		ObjectMeta: message.ObjectMeta{
			TypeMeta: scheme.TypeMeta{
				Kind:  "someTest",
				Group: "testGroup",
			},
		},
	}
	payload, err := json.Marshal(data)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("successfully process a pkg", func(t *testing.T) {
		incomingPkg := mockTransport.NewMockIncomingPkg(ctrl)
		incomingPkg.EXPECT().Payload().Return(payload)
		incomingPkg.EXPECT().UID().Return("123").Times(2)
		incomingPkg.EXPECT().Origin().Return("mb_topic")
		incomingPkg.EXPECT().Headers().Return(message.Headers{"traceId": "123", "uid": "1234"})

		marshaller.
			EXPECT().
			Unmarshal(payload).
			Return(data, nil)

		dispatcher.EXPECT().Match(data).Return([]execution.Executor{niceExecutor})

		err = pkgProcessor.Process(ctx, incomingPkg)
		assert.NoError(t, err)
	})

	t.Run("error unmarshalling payload", func(t *testing.T) {
		incomingPkg := mockTransport.NewMockIncomingPkg(ctrl)
		incomingPkg.EXPECT().Payload().Return(payload)
		marshaller.
			EXPECT().
			Unmarshal(payload).
			Return(nil, errors.New("some error"))

		err = pkgProcessor.Process(ctx, incomingPkg)
		assert.Error(t, err)
		assert.EqualError(t, err, "unmarshalling pkg payload: some error")
	})

	t.Run("no uid header found", func(t *testing.T) {
		incomingPkg := mockTransport.NewMockIncomingPkg(ctrl)
		incomingPkg.EXPECT().Payload().Return(payload)
		incomingPkg.EXPECT().UID().Return("")

		marshaller.
			EXPECT().
			Unmarshal(payload).
			Return(data, nil)

		err = pkgProcessor.Process(ctx, incomingPkg)
		assert.Error(t, err)
		assert.EqualError(t, err, "error finding uid header in received message. testGroup.someTest")
	})

	t.Run("no executors defined", func(t *testing.T) {
		incomingPkg := mockTransport.NewMockIncomingPkg(ctrl)
		incomingPkg.EXPECT().Payload().Return(payload)
		incomingPkg.EXPECT().UID().Return("123").Times(2)
		incomingPkg.EXPECT().Origin().Return("mb_topic")
		incomingPkg.EXPECT().Headers().Return(message.Headers{"traceId": "123", "uid": "1234"})

		marshaller.
			EXPECT().
			Unmarshal(payload).
			Return(data, nil)

		dispatcher.EXPECT().Match(data).Return([]execution.Executor{})

		err = pkgProcessor.Process(ctx, incomingPkg)
		assert.Error(t, err)
		assert.EqualError(t, err, "No executors defined for message uid 123 testGroup.someTest")
	})

	t.Run("executor returns an error", func(t *testing.T) {
		incomingPkg := mockTransport.NewMockIncomingPkg(ctrl)
		incomingPkg.EXPECT().Payload().Return(payload)
		incomingPkg.EXPECT().UID().Return("123").Times(2)
		incomingPkg.EXPECT().Origin().Return("mb_topic")
		incomingPkg.EXPECT().Headers().Return(message.Headers{"traceId": "123", "uid": "1234"})

		marshaller.
			EXPECT().
			Unmarshal(payload).
			Return(data, nil)

		dispatcher.EXPECT().Match(data).Return([]execution.Executor{executorWithError})

		err = pkgProcessor.Process(ctx, incomingPkg)
		assert.Error(t, err)
		assert.EqualError(t, err, "error executing message 123 testGroup.someTest: always return an error")
	})
}

// check ctx here instead of mocking MsgExecutionCtxFactory
func niceExecutor(execCtx execution.MessageExecutionCtx) error {
	traceIdVal := execCtx.Context().Value(ContextTraceIDKey)
	traceId, ok := traceIdVal.(string)
	if !ok {
		return errors.New("traceId value is not string")
	}

	if traceId == "" {
		return errors.New("traceId is empty")
	}

	return nil
}

func executorWithError(execCtx execution.MessageExecutionCtx) error {
	return errors.New("always return an error")
}
