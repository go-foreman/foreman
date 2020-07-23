package api

import (
	api "github.com/kopaygorodsky/brigadier/pkg/saga/api/handlers/status"
	"go.uber.org/fx"
)

var ModuleFx = fx.Options(
	fx.Provide(api.NewStatusHandler),
	fx.Provide(api.NewStatusService),
	fx.Invoke(StartServer),
)
