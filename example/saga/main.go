package main

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kopaygorodsky/brigadier/example/saga/handlers"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase/account/contracts"
	"github.com/kopaygorodsky/brigadier/pkg"
	"github.com/kopaygorodsky/brigadier/pkg/log"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	transportPackage "github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/plugins/amqp"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"github.com/kopaygorodsky/brigadier/pkg/saga"
	"github.com/kopaygorodsky/brigadier/pkg/saga/component"
	"github.com/kopaygorodsky/brigadier/pkg/saga/mutex"

	_ "github.com/kopaygorodsky/brigadier/example/saga/usecase/account"
)

var defaultLogger = log.DefaultLogger()

func main() {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/watcher?charset=utf8&parseTime=True&timeout=30s")
	if err != nil {
		panic(err)
	}
	db.SetMaxIdleConns(100)

	amqpTransport := amqp.NewTransport("amqp://admin:admin123@127.0.0.1:5672", defaultLogger)
	queue := amqp.Queue("messages", false, false, false, false)
	topic := amqp.Topic("messages_exchange", false, false, false, false)
	binds := amqp.QueueBind(topic.Name(), fmt.Sprintf("%s.#", topic.Name()), false)

	ctx := context.Background()

	if err := amqpTransport.Connect(ctx); err != nil {
		defaultLogger.Logf(log.ErrorLevel, "Error connecting to amqp. %s", err)
		panic(err)
	}

	if err := amqpTransport.CreateTopic(ctx, topic); err != nil {
		defaultLogger.Logf(log.ErrorLevel, "Error creating topic %s. %s", topic.Name(), err)
		panic(err)
	}

	if err := amqpTransport.CreateQueue(ctx, queue, binds); err != nil {
		defaultLogger.Logf(log.ErrorLevel, "Error creating queue %s. %s", queue.Name(), err)
		panic(err)
	}

	amqpEndpoint := endpoint.NewAmqpEndpoint(fmt.Sprintf("%s_endpoint", queue.Name()), amqpTransport, transportPackage.DeliveryDestination{DestinationTopic: topic.Name(), RoutingKey: fmt.Sprintf("%s.eventAndCommands", topic.Name())})

	sagaComponent := component.NewSagaComponent(
		func(scheme scheme.KnownTypesRegistry) (saga.Store, error) {
			return saga.NewSqlSagaStore(db, scheme)
		},
		mutex.NewSqlMutex(db),
	)
	sagaComponent.RegisterSagaEndpoints(amqpEndpoint)
	sagaComponent.RegisterSagas(usecase.DefaultSagasCollection.Sagas()...)
	sagaComponent.RegisterContracts(usecase.DefaultSagasCollection.Contracts()...)

	bus, err := pkg.NewMessageBus(defaultLogger, pkg.DefaultWithTransport(amqpTransport), pkg.WithSchemeRegistry(scheme.KnownTypesRegistryInstance), pkg.WithComponents(sagaComponent))

	if err != nil {
		panic(err)
	}

	accountHandler := handlers.NewAccountHandler(defaultLogger)
	//here goes registration of handlers
	bus.Dispatcher().RegisterCmdHandler(&contracts.RegisterAccountCmd{}, accountHandler.RegisterAccount)
	bus.Dispatcher().RegisterCmdHandler(&contracts.SendConfirmationCmd{}, accountHandler.SendConfirmation)

	defaultLogger.Log(log.PanicLevel, bus.Subscriber().Run(context.Background(), queue))
}
