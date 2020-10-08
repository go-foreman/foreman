package saga

import (
	"github.com/kopaygorodsky/brigadier/pkg"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"github.com/kopaygorodsky/brigadier/pkg/saga/contracts"
	"github.com/kopaygorodsky/brigadier/pkg/saga/handlers"
	"github.com/kopaygorodsky/brigadier/pkg/saga/mutex"
	logger "github.com/sirupsen/logrus"
)

type Module struct {
	sagas []Saga
	contracts []interface{}
	sagaStore Store
	sagaMutex mutex.Mutex
	schema scheme.KnownTypesRegistry
	endpoints []endpoint.Endpoint
	configOpts []ConfigOption
}

type opts struct {
	idExtractor IdExtractor
}

type ConfigOption func(o *opts)

func NewSagaModule(sagaStore Store, sagaMutex mutex.Mutex, schema scheme.KnownTypesRegistry, opts... ConfigOption) *Module {
	return &Module{sagaStore: sagaStore, sagaMutex: sagaMutex, schema: schema, configOpts: opts}
}

func (s Module) Init(bootstrapper *pkg.Bootstrapper) error {
	opts := &opts{}
	for _, config := range s.configOpts {
		config(opts)
	}

	if opts.idExtractor == nil {
		opts.idExtractor = NewSagaIdExtractor()
	}

	logrus := &logger.Logger{}
	eventHandler := handlers.NewEventsHandler(s.sagaStore, s.sagaMutex, s.schema,opts.idExtractor, logrus)
	sagaControlHandler := handlers.NewSagaControlHandler(s.sagaStore, s.sagaMutex, s.schema, logrus)

	msgDispatcher := bootstrapper.Dispatcher()
	msgDispatcher.RegisterCmdHandler(&contracts.StartSagaCommand{}, sagaControlHandler.Handle)
	msgDispatcher.RegisterCmdHandler(&contracts.RecoverSagaCommand{}, sagaControlHandler.Handle)
	msgDispatcher.RegisterCmdHandler(&contracts.CompensateSagaCommand{}, sagaControlHandler.Handle)

	for _, saga := range s.sagas {
		saga.Init()
		scheme.KnownTypesRegistryInstance.RegisterTypes(saga)

		for eventKey := range saga.EventHandlers() {
			msgDispatcher.RegisterEventListenerWithKey(scheme.WithKey(eventKey), eventHandler.Handle)
		}
	}

	for _, sagaEndpoint := range s.endpoints {
		bootstrapper.Router().RegisterEndpoint(sagaEndpoint,
			&contracts.StartSagaCommand{},
			&contracts.RecoverSagaCommand{},
			&contracts.CompensateSagaCommand{},
			&contracts.SagaCompletedEvent{},
			&contracts.SagaChildCompletedEvent{},
		)
	}

	return nil
}

func (s *Module) RegisterSagas(sagas... Saga) {
	s.sagas = append(s.sagas, sagas...)
}

func (s *Module) RegisterContracts(contracts... interface{}) {
	s.contracts = append(s.contracts, contracts...)
}

func (s *Module) RegisterSagaEndpoints(endpoints... endpoint.Endpoint) {
	s.endpoints = append(s.endpoints, endpoints...)
}

func WithSagaIdExtractor(extractor IdExtractor) ConfigOption {
	return func(o *opts) {
		o.idExtractor = extractor
	}
}

