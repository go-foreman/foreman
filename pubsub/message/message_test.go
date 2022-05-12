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
