package component

import (
	"github.com/kopaygorodsky/brigadier/pkg"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"github.com/kopaygorodsky/brigadier/pkg/saga"
	"github.com/kopaygorodsky/brigadier/pkg/saga/contracts"
	"github.com/kopaygorodsky/brigadier/pkg/saga/handlers"
	"github.com/kopaygorodsky/brigadier/pkg/saga/mutex"
)

type Component struct {
	sagas            []saga.Saga
	contracts        []interface{}
	sagaStoreFactory StoreFactory
	sagaMutex        mutex.Mutex
	schema           scheme.KnownTypesRegistry
	endpoints        []endpoint.Endpoint
	configOpts       []ConfigOption
}

type opts struct {
	idExtractor saga.IdExtractor
}

type ConfigOption func(o *opts)

func NewSagaComponent(sagaStoreFactory StoreFactory, sagaMutex mutex.Mutex, opts ...ConfigOption) *Component {
	return &Component{sagaStoreFactory: sagaStoreFactory, sagaMutex: sagaMutex, configOpts: opts}
}

func (c Component) Init(mBus *pkg.MessageBus) error {
	opts := &opts{}
	for _, config := range c.configOpts {
		config(opts)
	}

	if opts.idExtractor == nil {
		opts.idExtractor = saga.NewSagaIdExtractor()
	}

	store, err := c.sagaStoreFactory(mBus.SchemeRegistry())

	if err != nil {
		return err
	}

	eventHandler := handlers.NewEventsHandler(store, c.sagaMutex, c.schema, opts.idExtractor, mBus.Logger())
	sagaControlHandler := handlers.NewSagaControlHandler(store, c.sagaMutex, mBus.SchemeRegistry(), mBus.Logger())

	mBus.Dispatcher().RegisterCmdHandler(&contracts.StartSagaCommand{}, sagaControlHandler.Handle)
	mBus.Dispatcher().RegisterCmdHandler(&contracts.RecoverSagaCommand{}, sagaControlHandler.Handle)
	mBus.Dispatcher().RegisterCmdHandler(&contracts.CompensateSagaCommand{}, sagaControlHandler.Handle)

	for _, s := range c.sagas {
		s.Init()
		mBus.SchemeRegistry().RegisterTypes(s)

		for eventKey := range s.EventHandlers() {
			mBus.Dispatcher().RegisterEventListenerWithKey(scheme.WithKey(eventKey), eventHandler.Handle)
		}
	}

	for _, sagaEndpoint := range c.endpoints {
		mBus.Router().RegisterEndpoint(sagaEndpoint,
			&contracts.StartSagaCommand{},
			&contracts.RecoverSagaCommand{},
			&contracts.CompensateSagaCommand{},
			&contracts.SagaCompletedEvent{},
			&contracts.SagaChildCompletedEvent{},
		)
		mBus.Router().RegisterEndpoint(sagaEndpoint, c.contracts...)
	}

	return nil
}

func (c *Component) RegisterSagas(sagas ...saga.Saga) {
	c.sagas = append(c.sagas, sagas...)
}

func (c *Component) RegisterContracts(contracts ...interface{}) {
	c.contracts = append(c.contracts, contracts...)
}

func (c *Component) RegisterSagaEndpoints(endpoints ...endpoint.Endpoint) {
	c.endpoints = append(c.endpoints, endpoints...)
}

func WithSagaIdExtractor(extractor saga.IdExtractor) ConfigOption {
	return func(o *opts) {
		o.idExtractor = extractor
	}
}

type StoreFactory func(scheme scheme.KnownTypesRegistry) (saga.Store, error)
