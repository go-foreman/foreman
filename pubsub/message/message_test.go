package message

import (
	"testing"

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
