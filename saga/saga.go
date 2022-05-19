package saga

import (
	"fmt"
	"reflect"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
)

type Saga interface {
	// include Object interface as any message type in MessageBus a saga should have metadata
	message.Object
	// Init function assigns a contract type to a handler
	Init()
	// Start will be triggered when StartSagaCommand received
	Start(sagaCtx SagaContext) error
	// Compensate will be triggered when CompensateSagaCommand received
	Compensate(sagaCtx SagaContext) error
	// Recover will be triggered when RecoverSagaCommand received
	Recover(sagaCtx SagaContext) error
	// EventHandlers returns a list of assigned executors per type in Init()
	EventHandlers() map[scheme.GroupKind]Executor
	// SetSchema allows to set schema instance during the saga runtime
	SetSchema(scheme scheme.KnownTypesRegistry)
}

type BaseSaga struct {
	message.ObjectMeta
	adjacencyMap map[scheme.GroupKind]Executor
	scheme       scheme.KnownTypesRegistry
}

type Executor func(execCtx SagaContext) error

func (b *BaseSaga) AddEventHandler(ev message.Object, handler Executor) *BaseSaga {
	//lazy initialization
	if b.adjacencyMap == nil {
		b.adjacencyMap = make(map[scheme.GroupKind]Executor)
	}

	if b.scheme == nil {
		panic("schema wasn't set")
	}

	groupKind, err := b.scheme.ObjectKind(ev)

	if err != nil {
		panic(fmt.Sprintf("ev %s is not registered in schema", reflect.TypeOf(ev).String()))
	}

	b.adjacencyMap[*groupKind] = handler
	return b
}

func (b *BaseSaga) SetSchema(scheme scheme.KnownTypesRegistry) {
	b.scheme = scheme
}

func (b BaseSaga) EventHandlers() map[scheme.GroupKind]Executor {
	return b.adjacencyMap
}
