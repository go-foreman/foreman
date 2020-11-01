package execution

import (
	"context"
	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/endpoint"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/transport/pkg"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/pkg/errors"
	"time"
)

type MessageExecutionCtx interface {
	Message() *message.Message
	Context() context.Context
	Valid() bool
	Send(message *message.Message, options ...endpoint.DeliveryOption) error
	Return(delay time.Duration) error
	LogMessage(level log.Level, msg string)
}

type messageExecutionCtx struct {
	isValid bool
	ctx     context.Context
	inPkg   pkg.IncomingPkg
	message *message.Message
	router  endpoint.Router
	logger  log.Logger
}

func (m messageExecutionCtx) Valid() bool {
	return m.isValid
}

func (m messageExecutionCtx) Context() context.Context {
	return m.ctx
}

func (m messageExecutionCtx) Send(message *message.Message, options ...endpoint.DeliveryOption) error {
	endpoints := m.router.Route(scheme.WithKey(message.Name))

	if len(endpoints) == 0 {
		m.logger.Logf(log.WarnLevel, "No endpoints defined for message %s", message.Name)
		return nil
	}

	for _, endp := range endpoints {
		if err := endp.Send(m.ctx, message, message.Headers, options...); err != nil {
			m.logger.Logf(log.ErrorLevel, "Error sending message id %s", message.ID)
			return errors.WithStack(err)
		}
	}

	return nil
}

func (m messageExecutionCtx) Return(delay time.Duration) error {
	for {
		select {
		case <-m.ctx.Done():
			m.logger.Logf(log.InfoLevel, "Context is closed, exiting without returning msg: %s, delay is too long", m.message.ID)
			return nil
		case <-time.After(delay):
			m.message.Headers.RegisterReturn()
			if err := m.Send(m.message); err != nil {
				m.logger.Logf(log.ErrorLevel, "error when returning a message %s", m.message.ID)
				return errors.Wrapf(err, "error when returning a message %s", m.message.ID)
			}
		}
	}
}

func (m messageExecutionCtx) Message() *message.Message {
	return m.message
}

func (m messageExecutionCtx) LogMessage(lvl log.Level, msg string) {
	m.logger.Log(lvl, msg)
}

type MessageExecutionCtxFactory interface {
	CreateCtx(ctx context.Context, inPkg pkg.IncomingPkg, message *message.Message) MessageExecutionCtx
}

type messageExecutionCtxFactory struct {
	router endpoint.Router
	logger log.Logger
}

func NewMessageExecutionCtxFactory(router endpoint.Router, logger log.Logger) MessageExecutionCtxFactory {
	return &messageExecutionCtxFactory{router: router, logger: logger}
}

func (m messageExecutionCtxFactory) CreateCtx(ctx context.Context, inPkg pkg.IncomingPkg, message *message.Message) MessageExecutionCtx {
	return &messageExecutionCtx{ctx: ctx, inPkg: inPkg, message: message, router: m.router, logger: m.logger}
}

type NoDefinedEndpoints struct {
	error
}

func WithNoDefinedEndpoints(err error) error {
	return NoDefinedEndpoints{err}
}
