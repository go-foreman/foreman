package message

import (
	"time"

	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/google/uuid"
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
}

type ObjectMeta struct {
	scheme.TypeMeta
}

type ReceivedMessage struct {
	uid        string
	headers    Headers
	payload    Object
	receivedAt time.Time
	origin     string
}

func NewReceivedMessage(uid string, payload Object, headers Headers, receivedAt time.Time, origin string) *ReceivedMessage {
	return &ReceivedMessage{
		uid:        uid,
		headers:    headers,
		payload:    payload,
		receivedAt: receivedAt,
		origin:     origin,
	}
}

func (m ReceivedMessage) UID() string {
	return m.uid
}

func (m ReceivedMessage) TraceID() string {
	if m.headers == nil {
		return ""
	}

	traceIdVal, ok := m.headers["traceId"]

	if !ok {
		return ""
	}

	return traceIdVal.(string)
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
	obj     Object
	uid     string
	headers Headers
}

func (m OutcomingMessage) Headers() Headers {
	return m.headers
}

func (m OutcomingMessage) UID() string {
	return m.uid
}

func (m OutcomingMessage) Payload() Object {
	return m.obj
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

	msg := &OutcomingMessage{uid: uuid.New().String(), obj: payload}

	if opts.headers != nil {
		msg.headers = opts.headers
	}

	if msg.headers == nil {
		msg.headers = make(Headers)
	}

	msg.headers["uid"] = msg.UID()

	if opts.traceID != "" {
		msg.headers["traceId"] = opts.traceID
	}

	return msg
}

func FromReceivedMsg(received *ReceivedMessage) *OutcomingMessage {
	return &OutcomingMessage{
		headers: received.Headers(),
		uid:     received.UID(),
		obj:     received.Payload(),
	}
}

type MsgOption func(attr *opts)

type opts struct {
	headers Headers
	traceID string
}

func WithHeaders(headers Headers) MsgOption {
	return func(attr *opts) {
		attr.headers = headers
	}
}

func WithTraceID(traceID string) MsgOption {
	return func(attr *opts) {
		attr.traceID = traceID
	}
}
