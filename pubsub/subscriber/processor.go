package subscriber

import (
	"context"
	"fmt"

	"github.com/go-foreman/foreman/log"
	msgDispatcher "github.com/go-foreman/foreman/pubsub/dispatcher"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/pubsub/transport/pkg"
	"github.com/pkg/errors"
)

type Processor interface {
	Process(ctx context.Context, inPkg pkg.IncomingPkg) error
}

type processor struct {
	logger            log.Logger
	decoder           message.Decoder
	dispatcher        msgDispatcher.Dispatcher
	msgExecCtxFactory execution.MessageExecutionCtxFactory
}

func NewMessageProcessor(decoder message.Decoder, msgExecCtxFactory execution.MessageExecutionCtxFactory, msgDispatcher msgDispatcher.Dispatcher, logger log.Logger) Processor {
	return &processor{decoder: decoder, msgExecCtxFactory: msgExecCtxFactory, dispatcher: msgDispatcher, logger: logger}
}

func (p *processor) Process(ctx context.Context, inPkg pkg.IncomingPkg) error {
	msg, err := p.decoder.Decode(inPkg)
	if err != nil {
		p.logger.Logf(log.ErrorLevel, "Failed to decode IncomingPkg into Message. %s", err)
		return errors.WithStack(err)
	}

	if msg.Headers.ReturnsCount() >= 10 {
		return errors.Errorf("Message %s was returned more that 10 times. Not acking. It will be removed once TTL expires.", msg.ID)
	}

	executors := p.dispatcher.Match(msg)

	if len(executors) == 0 {
		errMsg := fmt.Sprintf("No executors defined for message %s %s of type %s.", msg.ID, msg.Name, msg.Type)
		p.logger.Log(log.ErrorLevel, errMsg)
		return WithNoExecutorsDefinedErr(errors.New(errMsg))
	}

	execCtx := p.msgExecCtxFactory.CreateCtx(ctx, inPkg, msg)

	for _, exec := range executors {
		if err := exec(execCtx); err != nil {
			return errors.Wrapf(err, "error executing message %s of type %s. %s.", msg.Name, msg.Type, err)
		}
	}

	return nil
}

type NoExecutorsDefinedErr struct {
	error
}

func WithNoExecutorsDefinedErr(err error) error {
	return &NoExecutorsDefinedErr{err}
}
