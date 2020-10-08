package pkg

import (
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/dispatcher"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
)

type Module interface {
	Init(bootstrapper *Bootstrapper) error
}

type Bootstrapper struct {
	messagesDispatcher dispatcher.Dispatcher
	router             endpoint.Router
	modules            []Module
}

func (b *Bootstrapper) ApplyModule(module Module) {
	b.modules = append(b.modules, module)
}

func (b *Bootstrapper) Dispatcher() dispatcher.Dispatcher {
	return b.messagesDispatcher
}

func (b *Bootstrapper) Router() endpoint.Router {
	return b.router
}
