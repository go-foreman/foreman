package execution

import (
	"context"

	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/endpoint"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../../../testing/mocks/pubsub/message/context.go -package execution . MessageExecutionCtx,MessageExecutionCtxFactory

// MessageExecutionCtx is passed to each executor and contains received message, ctx, knows how to send out or return a message.
type MessageExecutionCtx interface {
	// Message returns received message
	Message() *message.ReceivedMessage
	// Context returns parent execution context. Each message has own time limit in which it must be processed.
	Context() context.Context
	// Valid Deprecated
	Valid() bool
	// Send sends an out coming message to registered endpoints
	Send(message *message.OutcomingMessage, options ...endpoint.DeliveryOption) error
	// Return sends received message to registered endpoints and updates number of returns in headers
	Return(options ...endpoint.DeliveryOption) error
	// Logger returns logger instance with traceId and message uid included as fields
	Logger() log.Logger
}

type messageExecutionCtx struct {
	isValid bool
	ctx     context.Context
	message *message.ReceivedMessage
	router  endpoint.Router
	logger  log.Logger
}

func (m messageExecutionCtx) Valid() bool {
	return m.isValid
}

func (m messageExecutionCtx) Context() context.Context {
	return m.ctx
}

func (m messageExecutionCtx) Send(msg *message.OutcomingMessage, options ...endpoint.DeliveryOption) error {
	endpoints := m.router.Route(msg.Payload())

	if len(endpoints) == 0 {
		m.logger.Log(log.WarnLevel, "no endpoints defined for message")
		return nil
	}

	for _, endp := range endpoints {
		if err := endp.Send(m.ctx, msg, options...); err != nil {
			m.logger.Logf(log.ErrorLevel, "error sending message. %s", err)
			return errors.WithStack(err)
		}
	}

	return nil
}

func (m messageExecutionCtx) Return(options ...endpoint.DeliveryOption) error {
	outComingMsg := message.FromReceivedMsg(m.message)
	outComingMsg.Headers().RegisterReturn()
	if err := m.Send(outComingMsg, options...); err != nil {
		m.logger.Logf(log.ErrorLevel, "error when returning a message. %s", err)
		return errors.Wrapf(err, "returning message %s", outComingMsg.UID())
	}

	return nil
}

func (m messageExecutionCtx) Message() *message.ReceivedMessage {
	return m.message
}

func (m messageExecutionCtx) Logger() log.Logger {
	return m.logger
}

type MessageExecutionCtxFactory interface {
	CreateCtx(ctx context.Context, message *message.ReceivedMessage) MessageExecutionCtx
}

type messageExecutionCtxFactory struct {
	router endpoint.Router
	logger log.Logger
}

func NewMessageExecutionCtxFactory(router endpoint.Router, logger log.Logger) MessageExecutionCtxFactory {
	return &messageExecutionCtxFactory{router: router, logger: logger}
}

func (m messageExecutionCtxFactory) CreateCtx(ctx context.Context, message *message.ReceivedMessage) MessageExecutionCtx {
	fields := make([]log.Field, 1, 2)
	fields[0] = log.Field{Name: "uid", Val: message.UID()}

	if traceID := message.TraceID(); traceID != "" {
		fields = append(fields, log.Field{Name: "traceId", Val: traceID})
	}

	return &messageExecutionCtx{ctx: ctx, message: message, router: m.router, logger: m.logger.WithFields(fields)}
}

type NoDefinedEndpoints struct {
	error
}

func WithNoDefinedEndpoints(err error) error {
	return NoDefinedEndpoints{err}
}
