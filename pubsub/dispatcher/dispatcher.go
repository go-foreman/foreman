package dispatcher

import (
	"fmt"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/runtime/scheme"
	"reflect"
)

type Dispatcher interface {
	Match(obj message.Object) []execution.Executor
	SubscribeForCmd(obj message.Object, executor execution.Executor) Dispatcher
	SubscribeForEvent(obj message.Object, executor execution.Executor) Dispatcher
	SubscribeForAllEvents(executor execution.Executor) Dispatcher
}

func NewDispatcher() Dispatcher {
	return &dispatcher{
		handlers: make(map[reflect.Type][]execution.Executor),
		listeners: make(map[reflect.Type][]execution.Executor),
	}
}

type dispatcher struct {
	handlers  map[reflect.Type][]execution.Executor
	listeners map[reflect.Type][]execution.Executor
	allEvsListeners []execution.Executor
}

func (d dispatcher) Match(obj message.Object) []execution.Executor {
	structType := scheme.GetStructType(obj)
	handlers, exists := d.handlers[structType]

	if exists && len(handlers) > 0 {
		return handlers
	}

	listenersMap := make(map[uintptr]execution.Executor)

	for _, ev := range d.allEvsListeners {
		listenersMap[reflect.ValueOf(ev).Pointer()] = ev
	}

	eventListeners, exists := d.listeners[structType]

	if exists && len(eventListeners) > 0 {
		for _, ev := range eventListeners {
			listenersMap[reflect.ValueOf(ev).Pointer()] = ev
		}
	}

	allEvListeners := make([]execution.Executor, len(listenersMap))

	counter := 0
	for _, ev := range listenersMap {
		allEvListeners[counter] = ev
		counter++
	}

	return allEvListeners
}

func (d *dispatcher) SubscribeForCmd(obj message.Object, executor execution.Executor) Dispatcher {
	structType := scheme.GetStructType(obj)

	if _, subscribedForAnEvent := d.listeners[structType]; subscribedForAnEvent {
		panic(fmt.Sprintf("obj %s already subscribed for an event listener", structType.String()))
	}

	executorPtr := reflect.ValueOf(executor).Pointer()

	for _, handler := range d.handlers[structType] {
		handlerPtr := reflect.ValueOf(handler).Pointer()

		//check if this handler was already registered. because it's a function - need to take value and then ptr of it.
		if handlerPtr == executorPtr {
			return d
		}
	}

	d.handlers[structType] = append(d.handlers[structType], executor)
	return d
}

func (d *dispatcher) SubscribeForEvent(obj message.Object, executor execution.Executor) Dispatcher {
	structType := scheme.GetStructType(obj)

	if _, subscribedForACmd := d.handlers[structType]; subscribedForACmd {
		panic(fmt.Sprintf("obj %s already subscribed for a cmd handler", structType.String()))
	}

	executorPtr := reflect.ValueOf(executor).Pointer()

	for _, listener := range d.listeners[structType] {
		listenerPtr := reflect.ValueOf(listener).Pointer()

		//check if this listener was already registered
		if executorPtr == listenerPtr {
			return d
		}
	}

	d.listeners[structType] = append(d.listeners[structType], executor)
	return d
}

func (d *dispatcher) SubscribeForAllEvents(executor execution.Executor) Dispatcher {
	executorPtr := reflect.ValueOf(executor).Pointer()
	for _, listener := range d.allEvsListeners {
		listenerPtr := reflect.ValueOf(listener).Pointer()

		//check if this listener was already registered
		if executorPtr == listenerPtr {
			return d
		}
	}

	d.allEvsListeners = append(d.allEvsListeners, executor)
	return d
}
