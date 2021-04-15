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
	Message() *message.ReceivedMessage
	Context() context.Context
	Valid() bool
	Dispatch(payload message.Object, options ...endpoint.DeliveryOption)
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

func (s sagaCtx) Message() *message.ReceivedMessage {
	return s.execCtx.Message()
}

func (s sagaCtx) Context() context.Context {
	return s.execCtx.Context()
}

func (s sagaCtx) Valid() bool {
	return s.execCtx.Valid()
}

func (s sagaCtx) Return(options ...endpoint.DeliveryOption) {
	s.execCtx.Return()
	outcomingMsg := message.FromReceivedMsg(s.Message())
	outcomingMsg.Headers()[SagaIdKey] = s.sagaInstance.ID()
	s.execCtx.Return()
	s.Dispatch(outcomingMsg.Payload(), options...)
}

func (s sagaCtx) LogMessage(lvl log.Level, msg string) {
	s.execCtx.LogMessage(lvl, fmt.Sprintf("SagaId: %s :%s", s.sagaInstance.ID(), msg))
}

func (s sagaCtx) SagaInstance() Instance {
	return s.sagaInstance
}

func (s *sagaCtx) Dispatch(toDeliver message.Object, options ...endpoint.DeliveryOption) {
	s.deliveries = append(s.deliveries, &Delivery{
		Payload: toDeliver,
		Options: options,
	})
}

func (s sagaCtx) Deliveries() []*Delivery {
	return s.deliveries
}

type Delivery struct {
	Payload message.Object
	Options []endpoint.DeliveryOption
}
