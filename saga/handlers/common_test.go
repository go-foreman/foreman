package handlers

import (
	"github.com/go-foreman/foreman/pubsub/message"
	sagaPkg "github.com/go-foreman/foreman/saga"
)

type SagaExample struct {
	sagaPkg.BaseSaga
	Data           string
	err            error
	handleCallback func(sagaInst sagaPkg.Instance)
}

func (s *SagaExample) Init() {
	s.AddEventHandler(&DataContract{}, s.HandleData)
}

func (s *SagaExample) Start(sagaCtx sagaPkg.SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "start"})
	return s.err
}

func (s *SagaExample) Compensate(sagaCtx sagaPkg.SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "compensate"})
	return s.err
}

func (s *SagaExample) Recover(sagaCtx sagaPkg.SagaContext) error {
	if failedEv := sagaCtx.SagaInstance().Status().FailedOnEvent(); failedEv != nil {
		sagaCtx.Dispatch(failedEv)
		return s.err
	}

	sagaCtx.Dispatch(&DataContract{Message: "recover"})
	return s.err
}

func (s *SagaExample) HandleData(sagaCtx sagaPkg.SagaContext) error {
	sagaCtx.Dispatch(&DataContract{Message: "handle"})

	if s.handleCallback != nil {
		s.handleCallback(sagaCtx.SagaInstance())
	}

	return s.err
}

type DataContract struct {
	message.ObjectMeta
	Message string
}
