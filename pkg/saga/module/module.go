package module

import (
	"github.com/kopaygorodsky/brigadier/pkg/saga"
	"github.com/kopaygorodsky/brigadier/pkg/saga/handlers"
	"github.com/kopaygorodsky/brigadier/pkg/saga/mutex"
	"go.uber.org/fx"
)

var ModuleFx = fx.Options(
	fx.Provide(handlers.NewEventsHandler),
	fx.Provide(handlers.NewSagaStartedHandler),
	fx.Provide(NewMessageBusRegistrar),
	fx.Provide(saga.NewSagaIdExtractor),
	fx.Provide(saga.NewSqlSagaStore),
	fx.Provide(mutex.NewMysqlMutex),
)
