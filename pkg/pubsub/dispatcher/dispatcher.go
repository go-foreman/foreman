package dispatcher

import (
	"github.com/go-foreman/foreman/pkg/pubsub/message"
	"github.com/go-foreman/foreman/pkg/pubsub/message/execution"
	"github.com/go-foreman/foreman/pkg/runtime/scheme"
	"reflect"
)

const MatchesAllKeys = "*"

type Dispatcher interface {
	Match(message *message.Message) []execution.Executor
	RegisterCmdHandlerWithKey(cmdKeyChoice scheme.KeyChoice, executor execution.Executor) Dispatcher
	RegisterCmdHandler(cmd interface{}, executor execution.Executor) Dispatcher
	RegisterEventListenerWithKey(eventKeyChoice scheme.KeyChoice, executor execution.Executor) Dispatcher
	RegisterEventListener(event interface{}, executor execution.Executor) Dispatcher
}

func NewDispatcher() Dispatcher {
	return &dispatcher{handlers: make(map[string][]execution.Executor), listeners: make(map[string][]execution.Executor)}
}

type dispatcher struct {
	handlers  map[string][]execution.Executor
	listeners map[string][]execution.Executor
}

func (d dispatcher) Match(message *message.Message) []execution.Executor {
	if message.Type == "command" {
		if handlers, ok := d.handlers[message.Name]; ok {
			return handlers
		}
	} else if message.Type == "event" {

		var listeners []execution.Executor

		if catchAllListeners, ok := d.listeners[MatchesAllKeys]; ok {
			listeners = append(listeners, catchAllListeners...)
		}
		if eventListeners, ok := d.listeners[message.Name]; ok {
			listeners = append(listeners, eventListeners...)
		}

		return listeners
	}

	return nil
}

func (d *dispatcher) RegisterCmdHandlerWithKey(keyChoice scheme.KeyChoice, executor execution.Executor) Dispatcher {
	key := keyChoice()
	executorPtr := reflect.ValueOf(executor).Pointer()

	for _, handler := range d.handlers[key] {
		handlerPtr := reflect.ValueOf(handler).Pointer()

		//check if this handler was already registered. because it's a function - need to take value and then ptr of it.
		if handlerPtr == executorPtr {
			return d
		}
	}

	d.handlers[key] = append(d.handlers[key], executor)
	return d
}

func (d *dispatcher) RegisterEventListenerWithKey(keyChoice scheme.KeyChoice, executor execution.Executor) Dispatcher {
	key := keyChoice()
	executorPtr := reflect.ValueOf(executor).Pointer()
	for _, listener := range d.listeners[key] {
		listenerPtr := reflect.ValueOf(listener).Pointer()

		//check if this listener was already registered
		if executorPtr == listenerPtr {
			return d
		}
	}

	d.listeners[key] = append(d.listeners[key], executor)
	return d
}

func (d *dispatcher) RegisterCmdHandler(cmd interface{}, executor execution.Executor) Dispatcher {
	return d.RegisterCmdHandlerWithKey(scheme.WithStruct(cmd), executor)
}

func (d *dispatcher) RegisterEventListener(event interface{}, executor execution.Executor) Dispatcher {
	return d.RegisterEventListenerWithKey(scheme.WithStruct(event), executor)
}
