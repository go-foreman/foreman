package module

import (
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/dispatcher"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"github.com/kopaygorodsky/brigadier/pkg/saga"
	"github.com/kopaygorodsky/brigadier/pkg/saga/contracts"
	"github.com/kopaygorodsky/brigadier/pkg/saga/handlers"
)

type MessageBusRegistrar interface {
	RegisterSaga(saga saga.Saga)
}

type messageBusRegistrar struct {
	msgDispatcher dispatcher.Dispatcher
	eventExecutor *handlers.SagaEventsHandler
	router        endpoint.Router
}

func NewMessageBusRegistrar(msgDispatcher dispatcher.Dispatcher, router endpoint.Router, eventExecutor *handlers.SagaEventsHandler, starterSaga *handlers.SagaControlHandler) MessageBusRegistrar {
	msgDispatcher.RegisterCmdHandler(&contracts.StartSagaCommand{}, starterSaga.Handle)
	msgDispatcher.RegisterCmdHandler(&contracts.RecoverSagaCommand{}, starterSaga.Handle)
	msgDispatcher.RegisterCmdHandler(&contracts.CompensateSagaCommand{}, starterSaga.Handle)

	return &messageBusRegistrar{msgDispatcher: msgDispatcher, eventExecutor: eventExecutor, router: router}
}

func (m *messageBusRegistrar) RegisterSaga(saga saga.Saga) {
	saga.Init()
	scheme.KnownTypesRegistryInstance.RegisterTypes(saga)

	for eventKey := range saga.EventHandlers() {
		m.msgDispatcher.RegisterEventListenerWithKey(scheme.WithKey(eventKey), m.eventExecutor.Handle)
	}
}
