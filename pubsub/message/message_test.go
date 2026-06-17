package message

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SomeEvent struct {
	Object
	SomeData string
}

func TestNewOutcomingMessage(t *testing.T) {
	t.Run("basic constructor", func(t *testing.T) {
		ev := &SomeEvent{}
		m := NewOutcomingMessage(ev)

		assert.Equal(t, m.Payload(), ev)
		assert.NotEmpty(t, m.UID())
	})

	t.Run("with traceID", func(t *testing.T) {
		ev := &SomeEvent{}
		m := NewOutcomingMessage(ev, WithTraceID("sometraceid"), WithHeaders(Headers{"key": "val", "traceId": "this-will-be-overridden"}))
		assert.Equal(t, m.Payload(), ev)
		assert.NotEmpty(t, m.UID())
		assert.EqualValues(t, Headers{"traceId": "sometraceid", "key": "val", "uid": m.UID()}, m.Headers())
	})
}

func TestNewReceivedMessage(t *testing.T) {
	t.Run("basic constructor", func(t *testing.T) {
		ev := &SomeEvent{}
		m := NewReceivedMessage("uid", ev, Headers{"traceId": "xxx"}, time.Now(), "message_bus")
		assert.Equal(t, m.UID(), "uid")
		assert.Equal(t, m.TraceID(), "xxx")
		assert.Equal(t, m.Origin(), "message_bus")
	})

	t.Run("with nil trace id", func(t *testing.T) {
		ev := &SomeEvent{}
		m := NewReceivedMessage("uid", ev, Headers{"traceId": nil}, time.Now(), "message_bus")
		assert.Equal(t, m.TraceID(), "")
	})

	t.Run("with not existing trace id", func(t *testing.T) {
		ev := &SomeEvent{}
		m := NewReceivedMessage("uid", ev, Headers{}, time.Now(), "message_bus")
		assert.Equal(t, m.TraceID(), "")
	})
}

func TestReturnsCount(t *testing.T) {
	t.Run("absent header is zero", func(t *testing.T) {
		assert.Equal(t, 0, Headers{}.ReturnsCount())
	})

	t.Run("reads numeric types from any transport", func(t *testing.T) {
		assert.Equal(t, 3, Headers{"returnsCount": int(3)}.ReturnsCount())     // in-process
		assert.Equal(t, 3, Headers{"returnsCount": int32(3)}.ReturnsCount())   // AMQP round-trip
		assert.Equal(t, 3, Headers{"returnsCount": float64(3)}.ReturnsCount()) // JSON round-trip
	})

	t.Run("non-numeric is zero", func(t *testing.T) {
		assert.Equal(t, 0, Headers{"returnsCount": "oops"}.ReturnsCount())
	})
}

func TestRegisterReturn(t *testing.T) {
	t.Run("sets one when absent", func(t *testing.T) {
		h := Headers{}
		h.RegisterReturn()
		assert.Equal(t, 1, h.ReturnsCount())
	})

	t.Run("increments across transport type changes", func(t *testing.T) {
		h := Headers{"returnsCount": int32(1)} // value as it returns from AMQP
		h.RegisterReturn()
		assert.Equal(t, 2, h.ReturnsCount())
		h.RegisterReturn()
		assert.Equal(t, 3, h.ReturnsCount())
	})
}
