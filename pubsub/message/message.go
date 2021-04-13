package message

import (
	"github.com/go-foreman/foreman/runtime/scheme"
	"time"
)

type Headers map[string]interface{}

func (m Headers) ReturnsCount() int {
	v, exists := m["returnsCount"]
	if !exists {
		return 0
	}
	returnsCount, ok := v.(int)
	if !ok {
		return 0
	}
	return returnsCount
}

func (m Headers) RegisterReturn() {
	v, exists := m["returnsCount"]
	if !exists {
		m["returnsCount"] = 1
		return
	}

	returnsCount, ok := v.(int)
	if !ok {
		return
	}
	returnsCount++
	m["returnsCount"] = returnsCount
}

type Object interface {
	scheme.Object
	GetUID() string
}

type ObjectMeta struct {
	scheme.TypeMeta
	UID string `json:"uid"`
}

func (o ObjectMeta) GetUID() string {
	return o.UID
}

type ReceivedMessage struct {
	headers Headers
	payload Object
	receivedAt time.Time
	origin string
}

func (m ReceivedMessage) Headers() Headers {
	return m.headers
}

func (m ReceivedMessage) Payload() Object {
	return m.payload
}

func (m ReceivedMessage) ReceivedAt() time.Time {
	return m.receivedAt
}

func (m ReceivedMessage) Origin() string {
	return m.origin
}

type OutcomingMessage struct {
	headers Headers
	payload Object
}

func (m OutcomingMessage) Headers() Headers {
	return m.headers
}

func (m OutcomingMessage) Payload() Object {
	return m.payload
}

//NewOutcomingMessage accepts only structs as payload. If you want to pass scalar data type - wrap it in a struct.
func NewOutcomingMessage(payload Object, passedOptions ...MsgOption) *OutcomingMessage {
	opts := &opts{}

	if len(passedOptions) > 0 {
		for _, passedOpt := range passedOptions {
			if passedOpt != nil {
				passedOpt(opts)
			}
		}
	}

	msg := &OutcomingMessage{payload: payload}

	if opts.headers != nil {
		msg.headers = opts.headers
	}

	return msg
}

type MsgOption func(attr *opts)

type opts struct {
	description string
	headers     Headers
}

func WithHeaders(headers Headers) MsgOption {
	return func(attr *opts) {
		attr.headers = headers
	}
}
