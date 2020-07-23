package saga

import (
	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message/execution"
	"fmt"
	"github.com/pkg/errors"
	"time"
)

//SagaContext is sealed interface due to deliver method, that takes all dispatched deliveries and start sending out them
//Still not decided if this interface users should be able to implement
type SagaContext interface {
	//execution.MessageExecutionCtx
	Message() *message.Message
	Context() context.Context
	Valid() bool
	Dispatch(message *message.Message, options ...endpoint.DeliveryOption)
	Deliveries() []*Delivery
	Return(options ...endpoint.DeliveryOption)
	LogMessage(msg string, level string)
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

func (s sagaCtx) LogMessage(msg string, level string) {
	s.execCtx.LogMessage(fmt.Sprintf("SagaId: `%s`. %s", s.sagaInstance.ID(), msg), level)
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
			s.execCtx.LogMessage(fmt.Sprintf("error sending delivery for saga %s. Delivery: (%v). %s", s.SagaInstance().ID(), delivery, err), execution.LogError)
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
