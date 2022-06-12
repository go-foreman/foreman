package saga

import (
	"context"

	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/endpoint"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
)

//SagaContext is sealed interface due to deliver method, that takes all dispatched deliveries and start sending out them
//Still its' not decided if this interface users should be able to implement
type SagaContext interface {
	//execution.MessageExecutionCtx
	Message() *message.ReceivedMessage
	Context() context.Context
	// Valid Deprecated
	Valid() bool
	Dispatch(payload message.Object, options ...endpoint.DeliveryOption)
	Deliveries() []*Delivery
	Return(options ...endpoint.DeliveryOption) error
	Logger() log.Logger
	SagaInstance() Instance
}

func NewSagaCtx(execCtx execution.MessageExecutionCtx, sagaInstance Instance) SagaContext {
	return &sagaCtx{execCtx: execCtx, sagaInstance: sagaInstance, logger: execCtx.Logger().WithFields(log.Fields{sagaUIDKey: sagaInstance.UID()})}
}

type sagaCtx struct {
	logger       log.Logger
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

func (s sagaCtx) Return(options ...endpoint.DeliveryOption) error {
	s.Logger().Log(log.InfoLevel, "returning saga event")
	return s.execCtx.Return(options...)
}

func (s sagaCtx) Logger() log.Logger {
	return s.logger
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
