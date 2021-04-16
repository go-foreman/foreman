package dispatcher

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type registerAccountCmd struct {
	message.ObjectMeta
}

type sendConfirmationCmd struct {
	message.ObjectMeta
}

type accountRegisteredEvent struct {
	message.ObjectMeta
}

type confirmationSentEvent struct {
	message.ObjectMeta
}

func allExecutor(execCtx execution.MessageExecutionCtx) error {
	return nil
}

type service struct {
}

func (h *service) handle(execCtx execution.MessageExecutionCtx) error {
	return nil
}

func (h *service) anotherHandler(execCtx execution.MessageExecutionCtx) error {
	return nil
}

var handler = &service{}

func TestDispatcher_SubscribeForCmd(t *testing.T) {
	t.Run("subscribe for cmd by passing pointer to a struct", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForCmd(&registerAccountCmd{}, handler.handle)
		handlers := dispatcher.Match(&registerAccountCmd{})
		require.Len(t, handlers, 1)
		assertThisValueExists(t, handler.handle, handlers)
	})

	t.Run("multiple handlers for cmd", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForCmd(&sendConfirmationCmd{}, handler.handle)
		dispatcher.SubscribeForCmd(&sendConfirmationCmd{}, handler.anotherHandler)
		handlers := dispatcher.Match(&sendConfirmationCmd{})
		require.Len(t, handlers, 2)
		assertThisValueExists(t, handler.handle, handlers)
		assertThisValueExists(t, handler.anotherHandler, handlers)
	})

	t.Run("duplicate cmd - handler is ignored", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForCmd(&registerAccountCmd{}, handler.handle)
		handlers := dispatcher.Match(&registerAccountCmd{})
		require.Len(t, handlers, 1)
		assertThisValueExists(t, handler.handle, handlers)
	})

	t.Run("cmd is not struct type", func(t *testing.T) {
		dispatcher := NewDispatcher()
		wrongType := notStructType("aaa")
		assert.PanicsWithValue(t, "all types must be pointers to structs", func() {
			dispatcher.SubscribeForCmd(wrongType, handler.handle)
		})
	})
	
	t.Run("cross subscription for an event", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForCmd(&registerAccountCmd{}, handler.handle)
		assert.PanicsWithValue(t, "obj dispatcher.registerAccountCmd already subscribed for a cmd handler", func() {
			dispatcher.SubscribeForEvent(&registerAccountCmd{}, handler.handle)
		})
	})
}

func TestDispatcher_SubscribeForEvent(t *testing.T) {
	t.Run("subscribe for an event by passing pointer to a struct", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForEvent(&accountRegisteredEvent{}, handler.handle)
		listeners := dispatcher.Match(&accountRegisteredEvent{})
		require.Len(t, listeners, 1)
		assertThisValueExists(t, handler.handle, listeners)
	})

	t.Run("multiple listeners for an event", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForEvent(&accountRegisteredEvent{}, handler.handle)
		dispatcher.SubscribeForEvent(&accountRegisteredEvent{}, handler.anotherHandler)
		listeners := dispatcher.Match(&accountRegisteredEvent{})
		require.Len(t, listeners, 2)
		assertThisValueExists(t, handler.handle, listeners)
		assertThisValueExists(t, handler.anotherHandler, listeners)
	})

	t.Run("duplicate event - listener is ignored", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForEvent(&accountRegisteredEvent{}, handler.handle)
		dispatcher.SubscribeForEvent(&accountRegisteredEvent{}, handler.handle)
		listeners := dispatcher.Match(&accountRegisteredEvent{})
		require.Len(t, listeners, 1)
		assertThisValueExists(t, handler.handle, listeners)
	})

	t.Run("event is not struct type", func(t *testing.T) {
		dispatcher := NewDispatcher()
		wrongType := notStructType("aaa")
		assert.Panics(t, func() {
			dispatcher.SubscribeForCmd(wrongType, handler.handle)
		})
	})

	t.Run("cross subscription for a cmd", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForEvent(&accountRegisteredEvent{}, handler.handle)
		assert.PanicsWithValue(t, "obj dispatcher.accountRegisteredEvent already subscribed for an event listener", func() {
			dispatcher.SubscribeForCmd(&accountRegisteredEvent{}, handler.handle)
		})
	})
}

func TestDispatcher_SubscribeForAllEvents(t *testing.T) {
	t.Run("subscribe for all events, event those which weren't registered", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForAllEvents(handler.handle)
		listeners := dispatcher.Match(&accountRegisteredEvent{})
		require.Len(t, listeners, 1)
		assertThisValueExists(t, handler.handle, listeners)
	})
	
	t.Run("subscribe for an event by duplicating handler for all events", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForAllEvents(handler.handle)
		dispatcher.SubscribeForEvent(&accountRegisteredEvent{}, handler.handle)
		listeners := dispatcher.Match(&accountRegisteredEvent{})
		require.Len(t, listeners, 1)
		assertThisValueExists(t, handler.handle, listeners)
	})

	t.Run("subscribe for an event + match all events listeners", func(t *testing.T) {
		dispatcher := NewDispatcher()
		dispatcher.SubscribeForAllEvents(handler.handle)
		dispatcher.SubscribeForEvent(&confirmationSentEvent{}, handler.anotherHandler)
		listeners := dispatcher.Match(&confirmationSentEvent{})
		require.Len(t, listeners, 2)
		assertThisValueExists(t, handler.handle, listeners)
		assertThisValueExists(t, handler.anotherHandler, listeners)
	})
}

type notStructType string

func (n notStructType) GroupKind() scheme.GroupKind {
	panic("implement me")
}

func (n notStructType) SetGroupKind(gk *scheme.GroupKind) {

}

func assertThisValueExists(t *testing.T, expected execution.Executor, executors []execution.Executor) {
	exists := false
	for _, e := range executors {
		expectedPtr := reflect.ValueOf(expected).Pointer()
		currentPtr := reflect.ValueOf(e).Pointer()
		if expectedPtr == currentPtr {
			exists = true
		}
	}
	assert.True(t, exists, "expected executor is not found among executors")
}

