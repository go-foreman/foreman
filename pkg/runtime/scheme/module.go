package scheme

import "go.uber.org/fx"

var ModuleFx = fx.Options(
	fx.Provide(func() KnownTypesRegistry {
		return KnownTypesRegistryInstance
	}),
)
