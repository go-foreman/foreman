package saga

import (
	"context"
	"fmt"
	"time"

	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/endpoint"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/pkg/errors"
)

//SagaContext is sealed interface due to deliver method, that takes all dispatched deliveries and start sending out them
//Still its' not decided if this interface users should be able to implement
type SagaContext interface {
	//execution.MessageExecutionCtx
	Message() *message.Message
	Context() context.Context
	Valid() bool
	Dispatch(message *message.Message, options ...endpoint.DeliveryOption)
	Deliveries() []*Delivery
	Return(options ...endpoint.DeliveryOption)
	LogMessage(level log.Level, msg string)
	SagaInstance() Instance
	deliver() error
}

func NewSagaCtx(execCtx execution.MessageExecutionCtx, sagaInstance Instance) SagaContext {
	return &sagaCtx{execCtx: execCtx, sagaInstance: sagaInstance}
}

type sagaCtx struct {
	execCtx      execution.MessageExecutionCtx
	sagaInstance Instance
	deliveries   []*Delivery
}

func (s sagaCtx) Message() *message.Message {
	return s.execCtx.Message()
}

func (s sagaCtx) Context() context.Context {
	return s.execCtx.Context()
}

func (s sagaCtx) Valid() bool {
	return s.execCtx.Valid()
}

func (s sagaCtx) Return(options ...endpoint.DeliveryOption) {
	s.Dispatch(s.Message(), options...)
}

func (s sagaCtx) LogMessage(lvl log.Level, msg string) {
	s.execCtx.LogMessage(lvl, fmt.Sprintf("SagaId: %s :%s", s.sagaInstance.ID(), msg))
}

func (s sagaCtx) SagaInstance() Instance {
	return s.sagaInstance
}

func (s *sagaCtx) Dispatch(toDeliver *message.Message, options ...endpoint.DeliveryOption) {
	toDeliver.Headers[SagaIdKey] = s.sagaInstance.ID()
	s.deliveries = append(s.deliveries, &Delivery{
		Message: toDeliver,
		Options: options,
	})
}

func (s sagaCtx) deliver() error {
	for _, delivery := range s.Deliveries() {
		if err := s.execCtx.Send(delivery.Message, delivery.Options...); err != nil {
			s.execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("error sending delivery for saga %s. Delivery: (%v). %s", s.SagaInstance().ID(), delivery, err))
			return errors.Wrapf(err, "error sending delivery for saga %s. Delivery: (%v)", s.SagaInstance().ID(), delivery)
		}
		s.SagaInstance().AttachEvent(HistoryEvent{Metadata: delivery.Message.Metadata, Payload: delivery.Message.Payload, CreatedAt: time.Now()})
	}

	return nil
}

func (s sagaCtx) Deliveries() []*Delivery {
	return s.deliveries
}

type Delivery struct {
	Message *message.Message
	Options []endpoint.DeliveryOption
}
