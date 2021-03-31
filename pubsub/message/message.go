package message

import (
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"time"
)

const (
	EventType MessageType = "event"
	CommandType MessageType = "command"
)

type Metadata struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Type    MessageType  `json:"type"`
	Headers Headers `json:"headers"`
}

func (m *Message) ReturnsCount() int {
	v, exists := m.Headers["returnsCount"]
	if !exists {
		return 0
	}
	returnsCount, ok := v.(int)
	if !ok {
		return 0
	}
	return returnsCount
}

type Message struct {
	Metadata     `json:"metadata"`
	Payload      interface{} `json:"payload"`
	Description  string      `json:"description"`
	OriginSource string      `json:"-"`
	ReceivedAt   time.Time   `json:"-"`
}

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

type MessageType string

func ParseMessageType(msgType string) (MessageType, error) {
	var t MessageType
	switch msgType {
	case "event":
		t = EventType
	case "command":
		t = CommandType
	default:
		return "", errors.Errorf("unknown messageType")
	}

	return t, nil
}

func NewEventMessage(payload interface{}, options ...MsgOption) *Message {
	return NewMessage(scheme.WithStruct(payload), EventType, payload, options...)
}

func NewCommandMessage(payload interface{}, options ...MsgOption) *Message {
	return NewMessage(scheme.WithStruct(payload), CommandType, payload, options...)
}

//NewMessage accepts only structs as payload. If you want to pass scalar data type - wrap it in a struct.
func NewMessage(keyChoice scheme.KeyChoice, msgType MessageType, payload interface{}, passedOptions ...MsgOption) *Message {
	opts := &opts{}

	if len(passedOptions) > 0 {
		for _, passedOpt := range passedOptions {
			if passedOpt != nil {
				passedOpt(opts)
			}
		}
	}

	msg := &Message{Metadata: Metadata{ID: uuid.New().String(), Name: keyChoice(), Type: msgType, Headers: make(Headers)}, Payload: payload}

	if opts.description != "" {
		msg.Description = opts.description
	}

	if opts.headers != nil {
		msg.Headers = opts.headers
	}

	return msg
}

type MsgOption func(attr *opts)

type opts struct {
	description string
	headers     Headers
}

func WithDescription(description string) MsgOption {
	return func(attr *opts) {
		attr.description = description
	}
}

func WithHeaders(headers Headers) MsgOption {
	return func(attr *opts) {
		attr.headers = headers
	}
}
