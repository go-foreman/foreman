package execution

import (
	"context"
	"fmt"

	"github.com/go-foreman/foreman/pubsub/transport"

	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/endpoint"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"
)

type MessageExecutionCtx interface {
	Message() *message.ReceivedMessage
	Context() context.Context
	Valid() bool
	Send(message *message.OutcomingMessage, options ...endpoint.DeliveryOption) error
	Return(options ...endpoint.DeliveryOption) error
	LogMessage(level log.Level, msg string)
}

type messageExecutionCtx struct {
	isValid bool
	ctx     context.Context
	inPkg   transport.IncomingPkg
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
		m.logger.Logf(log.WarnLevel, "no endpoints defined for message %s", msg.UID())
		return nil
	}

	for _, endp := range endpoints {
		if err := endp.Send(m.ctx, msg, options...); err != nil {
			m.logger.Logf(log.ErrorLevel, "error sending message id %s", msg.UID())
			return errors.WithStack(err)
		}
	}

	return nil
}

func (m messageExecutionCtx) Return(options ...endpoint.DeliveryOption) error {
	outComingMsg := message.FromReceivedMsg(m.message)
	m.message.Headers().RegisterReturn()
	if err := m.Send(outComingMsg, options...); err != nil {
		m.logger.Logf(log.ErrorLevel, "error when returning a message %s", outComingMsg.UID())
		return errors.Wrapf(err, "returning message %s", outComingMsg.UID())
	}

	return nil
}

func (m messageExecutionCtx) Message() *message.ReceivedMessage {
	return m.message
}

func (m messageExecutionCtx) LogMessage(lvl log.Level, msg string) {
	if m.Message().TraceID() != "" {
		m.logger.Log(lvl, fmt.Sprintf("TraceID: %s : %s", m.Message().TraceID(), msg))
		return
	}

	m.logger.Log(lvl, msg)
}

type MessageExecutionCtxFactory interface {
	CreateCtx(ctx context.Context, inPkg transport.IncomingPkg, message *message.ReceivedMessage) MessageExecutionCtx
}

type messageExecutionCtxFactory struct {
	router endpoint.Router
	logger log.Logger
}

func NewMessageExecutionCtxFactory(router endpoint.Router, logger log.Logger) MessageExecutionCtxFactory {
	return &messageExecutionCtxFactory{router: router, logger: logger}
}

func (m messageExecutionCtxFactory) CreateCtx(ctx context.Context, inPkg transport.IncomingPkg, message *message.ReceivedMessage) MessageExecutionCtx {
	return &messageExecutionCtx{ctx: ctx, inPkg: inPkg, message: message, router: m.router, logger: m.logger}
}

type NoDefinedEndpoints struct {
	error
}

func WithNoDefinedEndpoints(err error) error {
	return NoDefinedEndpoints{err}
}
