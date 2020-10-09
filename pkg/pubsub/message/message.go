package message

import (
	"github.com/google/uuid"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"time"
)

type Metadata struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	Headers Headers `json:"headers"`
}

type Message struct {
	Metadata     `json:"metadata"`
	Payload      interface{} `json:"payload"`
	Description  string      `json:"description"`
	OriginSource string      `json:"-"`
	ReceivedAt   time.Time   `json:"-"`
}

type Headers map[string]interface{}

func NewEventMessage(payload interface{}, options ...MsgOption) *Message {
	return NewMessage(scheme.WithStruct(payload), "event", payload, options...)
}

func NewCommandMessage(payload interface{}, options ...MsgOption) *Message {
	return NewMessage(scheme.WithStruct(payload), "command", payload, options...)
}

//NewMessage accepts only structs as payload. If you want to pass scalar data type - wrap it in a struct.
func NewMessage(keyChoice scheme.KeyChoice, msgType string, payload interface{}, passedOptions ...MsgOption) *Message {
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
