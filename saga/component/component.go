package component

import (
	"github.com/go-foreman/foreman"
	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/endpoint"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/saga"
	"github.com/go-foreman/foreman/saga/api/handlers/status"
	"github.com/go-foreman/foreman/saga/contracts"
	"github.com/go-foreman/foreman/saga/handlers"
	"github.com/go-foreman/foreman/saga/mutex"
	"github.com/pkg/errors"
	"net/http"
)

type Component struct {
	sagas            []saga.Saga
	contracts        []message.Object
	sagaStoreFactory StoreFactory
	sagaMutex        mutex.Mutex
	endpoints        []endpoint.Endpoint
	configOpts       []configOption
}

type opts struct {
	uidService   saga.SagaUIDService
	apiServerMux *http.ServeMux
}

type configOption func(o *opts)

func NewSagaComponent(sagaStoreFactory StoreFactory, sagaMutex mutex.Mutex, opts ...configOption) *Component {
	return &Component{sagaStoreFactory: sagaStoreFactory, sagaMutex: sagaMutex, configOpts: opts}
}

func (c Component) Init(mBus *brigadier.MessageBus) error {
	opts := &opts{}
	for _, config := range c.configOpts {
		config(opts)
	}

	if opts.uidService == nil {
		opts.uidService = saga.NewSagaUIDService()
	}

	store, err := c.sagaStoreFactory(mBus.Marshaller())

	if err != nil {
		return err
	}

	if opts.apiServerMux != nil {
		initApiServer(opts.apiServerMux, store, mBus.Logger())
	}

	eventHandler := handlers.NewEventsHandler(store, c.sagaMutex, mBus.SchemeRegistry(), opts.uidService, mBus.Logger())
	sagaControlHandler := handlers.NewSagaControlHandler(store, c.sagaMutex, mBus.SchemeRegistry(), opts.uidService, mBus.Logger())

	contracts.RegisterSagaContracts(mBus.SchemeRegistry())

	mBus.Dispatcher().SubscribeForCmd(&contracts.StartSagaCommand{}, sagaControlHandler.Handle)
	mBus.Dispatcher().SubscribeForCmd(&contracts.RecoverSagaCommand{}, sagaControlHandler.Handle)
	mBus.Dispatcher().SubscribeForCmd(&contracts.CompensateSagaCommand{}, sagaControlHandler.Handle)

	for _, s := range c.sagas {
		s.SetSchema(mBus.SchemeRegistry())
		s.Init()

		for evGK := range s.EventHandlers() {
			//event obj must be registered in schema before
			evObj, err := mBus.SchemeRegistry().NewObject(evGK)
			if err != nil {
				return errors.Errorf("error creating an event object from scheme GK %s", evGK.String())
			}

			mBus.Dispatcher().SubscribeForEvent(evObj, eventHandler.Handle)
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

func (c *Component) RegisterContracts(contracts ...message.Object) {
	c.contracts = append(c.contracts, contracts...)
}

func (c *Component) RegisterSagaEndpoints(endpoints ...endpoint.Endpoint) {
	c.endpoints = append(c.endpoints, endpoints...)
}

func WithSagaUIDService(svc saga.SagaUIDService) configOption {
	return func(o *opts) {
		o.uidService = svc
	}
}

func WithSagaApiServer(mux *http.ServeMux) configOption {
	return func(o *opts) {
		o.apiServerMux = mux
	}
}

func initApiServer(mux *http.ServeMux, store saga.Store, logger log.Logger) {
	statusHandler := status.NewStatusHandler(logger, status.NewStatusService(store))
	mux.HandleFunc("/sagas", statusHandler.GetFilteredBy)
	mux.HandleFunc("/sagas/", statusHandler.GetStatus)
}

type StoreFactory func(msgMarshaller message.Marshaller) (saga.Store, error)
