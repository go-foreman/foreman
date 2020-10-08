package subscriber

import (
	"context"
	"github.com/kopaygorodsky/brigadier/pkg/log"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/dispatcher"
	pubsubErrs "github.com/kopaygorodsky/brigadier/pkg/pubsub/errors"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message/execution"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	"github.com/pkg/errors"
	"time"
)

type Processor interface {
	Process(ctx context.Context, inPkg pkg.IncomingPkg) error
}

type processor struct {
	logger            log.Logger
	decoder           message.Decoder
	dispatcher        dispatcher.Dispatcher
	msgExecCtxFactory execution.MessageExecutionCtxFactory
}

func NewMessageProcessor(decoder message.Decoder, msgExecCtxFactory execution.MessageExecutionCtxFactory, msgDispatcher dispatcher.Dispatcher, logger log.Logger) Processor {
	return &processor{decoder: decoder, msgExecCtxFactory: msgExecCtxFactory, dispatcher: msgDispatcher, logger: logger}
}

func (p *processor) Process(ctx context.Context, inPkg pkg.IncomingPkg) error {
	msg, err := p.decoder.Decode(inPkg)
	if err != nil {
		p.logger.Logf(log.ErrorLevel, "Failed to decode IncomingPkg into Message. %s", err)
		return errors.WithStack(err)
	}

	executors := p.dispatcher.Match(msg)

	if len(executors) == 0 {
		p.logger.Logf(log.ErrorLevel, "No executors defined for message %s of type %s", msg.Name, msg.Type)
		return WithNoExecutorsDefinedErr(errors.Errorf("No executors defined for message %s of type %s", msg.Name, msg.Type))
	}

	execCtx := p.msgExecCtxFactory.CreateCtx(ctx, inPkg, msg)

	for _, exec := range executors {
		if err := exec(execCtx); err != nil {
			p.logger.Logf(log.ErrorLevel, "Error executing message %s of type %s. %s", msg.Name, msg.Type, err)
			originalErr := errors.Cause(err)

			if statusErr, ok := originalErr.(pubsubErrs.StatusErr); ok {
				switch statusErr.Status {
				case pubsubErrs.NoRetry:
					p.logger.Logf(log.ErrorLevel, "Error executing message %s of type %s. %s. NoRetry", msg.Name, msg.Type, err)
				default:
					p.logger.Logf(log.ErrorLevel, "Error executing message %s of type %s. %s", msg.Name, msg.Type, err)
					return execCtx.Return(time.Second * 3)
				}
			}

			return errors.Wrapf(err, "Error executing message %s of type %s. %s", msg.Name, msg.Type, err)
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
