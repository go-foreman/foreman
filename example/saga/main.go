package saga

import (
	"context"
	transportPackage "github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	logger "github.com/sirupsen/logrus"

	"fmt"
	"github.com/kopaygorodsky/brigadier/pkg"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/plugins/amqp"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"github.com/kopaygorodsky/brigadier/pkg/saga"
	"github.com/kopaygorodsky/brigadier/pkg/saga/mutex"
)

var defaultLogger = &logger.Logger{}

func main() {
	sagaStore, err := saga.NewSqlSagaStore(nil, scheme.KnownTypesRegistryInstance)
	if err != nil {
		panic(err)
	}

	amqpTransport := amqp.NewTransport("amqp://admin:admin123@127.0.0.1:5672", defaultLogger)
	queue := amqp.Queue("messages", false, false, false, false)
	topic := amqp.Topic("messages_exchange", false, false, false, false)
	binds := amqp.QueueBind(topic.Name(), fmt.Sprintf("%s.#", topic.Name()), false)

	ctx := context.Background()

	if err := amqpTransport.Connect(ctx); err != nil {
		defaultLogger.Errorf("Error connecting to amqp. %s", err)
		panic(err)
	}

	if err := amqpTransport.CreateTopic(ctx, topic); err != nil {
		defaultLogger.Errorf("Error creating topic %s. %s", topic.Name(), err)
		panic(err)
	}

	if err := amqpTransport.CreateQueue(ctx, queue, binds); err != nil {
		defaultLogger.Errorf("Error creating queue %s. %s", queue.Name(), err)
		panic(err)
	}

	amqpEndpoint := endpoint.NewAmqpEndpoint(fmt.Sprintf("%s_endpoint", queue.Name()), amqpTransport, transportPackage.DeliveryDestination{DestinationTopic: topic.Name(), RoutingKey: fmt.Sprintf("%s.eventAndCommands", topic.Name())})

	sagaModule := saga.NewSagaModule(sagaStore, mutex.NewSqlMutex(nil), scheme.KnownTypesRegistryInstance, saga.WithSagaIdExtractor(saga.NewSagaIdExtractor()))
	sagaModule.RegisterSagaEndpoints(amqpEndpoint)
	sagaModule.RegisterSagas(DefaultSagasCollection.Sagas()...)
	sagaModule.RegisterContracts(DefaultSagasCollection.Contracts()...)

	bootstrapper := &pkg.Bootstrapper{}
	bootstrapper.ApplyModule(sagaModule)
}
