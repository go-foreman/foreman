package subscriber

import (
	"context"
	"fmt"
	"time"

	"github.com/go-foreman/foreman/pubsub/transport"

	"github.com/go-foreman/foreman/log"
	msgDispatcher "github.com/go-foreman/foreman/pubsub/dispatcher"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/pkg/errors"
)

const (
	ContextTraceIDKey PackageProperty = "traceId"
)

//go:generate mockgen --build_flags=--mod=mod -destination ../../testing/mocks/pubsub/subscriber/processor.go -package subscriber . Processor

// Processor knows how to process a message received by subscriber
type Processor interface {
	Process(ctx context.Context, inPkg transport.IncomingPkg) error
}

type processor struct {
	logger            log.Logger
	decoder           message.Marshaller
	dispatcher        msgDispatcher.Dispatcher
	msgExecCtxFactory execution.MessageExecutionCtxFactory
}

// NewMessageProcessor returns default implementation of Processor
func NewMessageProcessor(decoder message.Marshaller, msgExecCtxFactory execution.MessageExecutionCtxFactory, msgDispatcher msgDispatcher.Dispatcher, logger log.Logger) Processor {
	return &processor{decoder: decoder, msgExecCtxFactory: msgExecCtxFactory, dispatcher: msgDispatcher, logger: logger}
}

func (p *processor) Process(ctx context.Context, inPkg transport.IncomingPkg) error {
	payload, err := p.decoder.Unmarshal(inPkg.Payload())
	if err != nil {
		p.logger.Logf(log.ErrorLevel, "Failed to decode IncomingPkg into Message. %s", err)
		return errors.Wrap(err, "unmarshalling pkg payload")
	}

	if inPkg.UID() == "" {
		return errors.Errorf("error finding uid header in received message. %s", payload.GroupKind().String())
	}

	receivedMsg := message.NewReceivedMessage(inPkg.UID(), payload, inPkg.Headers(), time.Now(), inPkg.Origin())

	executors := p.dispatcher.Match(payload)

	if len(executors) == 0 {
		errMsg := fmt.Sprintf("No executors defined for message uid %s %s", receivedMsg.UID(), payload.GroupKind())
		p.logger.Log(log.ErrorLevel, errMsg)
		return WithNoExecutorsDefinedErr(errors.New(errMsg))
	}

	if traceID := receivedMsg.TraceID(); traceID != "" {
		ctx = context.WithValue(ctx, ContextTraceIDKey, traceID)
	}

	execCtx := p.msgExecCtxFactory.CreateCtx(ctx, receivedMsg)

	for _, exec := range executors {
		if err := exec(execCtx); err != nil {
			return errors.Wrapf(err, "error executing message %s %s", receivedMsg.UID(), payload.GroupKind())
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

type PackageProperty string
