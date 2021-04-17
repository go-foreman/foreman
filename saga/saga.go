package saga

import (
	"fmt"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	"reflect"
)

type Saga interface {
	message.Object
	Init()
	Start(sagaCtx SagaContext) error
	Compensate(sagaCtx SagaContext) error
	Recover(sagaCtx SagaContext) error
	EventHandlers() map[scheme.GroupKind]Executor
	SetSchema(scheme scheme.KnownTypesRegistry)
}

type BaseSaga struct {
	message.ObjectMeta
	adjacencyMap map[scheme.GroupKind]Executor
	scheme scheme.KnownTypesRegistry
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
