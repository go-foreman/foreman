package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kopaygorodsky/brigadier/example/saga/handlers"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase"
	_ "github.com/kopaygorodsky/brigadier/example/saga/usecase/account"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase/account/contracts"
	"github.com/kopaygorodsky/brigadier/pkg"
	"github.com/kopaygorodsky/brigadier/pkg/log"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/endpoint"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	transportPackage "github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/plugins/amqp"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"github.com/kopaygorodsky/brigadier/pkg/saga"
	"github.com/kopaygorodsky/brigadier/pkg/saga/component"
	"github.com/kopaygorodsky/brigadier/pkg/saga/mutex"
	amqp2 "github.com/streadway/amqp"
	"io/ioutil"
	"os"
	"time"
)

const (
	queueName = "messagebus"
	topicName = "messagebus_exchange"
)

var defaultLogger = log.DefaultLogger()

func main() {
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/watcher?charset=utf8&parseTime=True&timeout=30s")
	handleErr(err)
	db.SetMaxIdleConns(100)

	amqpTransport := amqp.NewTransport("amqp://admin:admin123@127.0.0.1:5672", defaultLogger)
	queue := amqp.Queue(queueName, false, false, false, false)
	topic := amqp.Topic(topicName, false, false, false, false)
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

	handleErr(err)

	//messagebus is ready to be used
	//here goes loading of DI container with all handlers, business entities etc
	loadSomeDIContainer(bus, defaultLogger)

	//run subscriber
	defaultLogger.Log(log.PanicLevel, bus.Subscriber().Run(context.Background(), queue))
}

func loadSomeDIContainer(bus *pkg.MessageBus, defaultLogger log.Logger) {
	tmpDir, err := ioutil.TempDir("", "confirmations")
	handleErr(err)
	accountHandler, err := handlers.NewAccountHandler(defaultLogger, tmpDir)
	handleErr(err)
	//here goes registration of handlers
	bus.Dispatcher().RegisterCmdHandler(&contracts.RegisterAccountCmd{}, accountHandler.RegisterAccount)
	bus.Dispatcher().RegisterCmdHandler(&contracts.SendConfirmationCmd{}, accountHandler.SendConfirmation)

	//and here we are going to run some application that will simulate user behaviour. Look into mailbox, click confirm and complete registration.
	go watchAndConfirmRegistration(tmpDir, defaultLogger)
}

func watchAndConfirmRegistration(dir string, logger log.Logger) {
	amqpConnection, err := amqp2.Dial("amqp://admin:admin123@127.0.0.1:5672")
	handleErr(err)

	amqpChannel, err := amqpConnection.Channel()
	handleErr(err)

	defer amqpConnection.Close()
	defer amqpChannel.Close()
	defer os.RemoveAll(dir)

	for {
		select {
		case <- time.After(time.Second * 5):
			files, err := ioutil.ReadDir(dir)
			handleErr(err)

			for _, info := range files {
				time.Sleep(time.Second)
				handleErr(err)
				filePath := dir+"/"+info.Name()
				uid, err := ioutil.ReadFile(filePath)
				handleErr(err)
				accountConfirmedEvent := &contracts.AccountConfirmed{UID: string(uid)}
				msgToDeliver := message.NewEventMessage(accountConfirmedEvent)
				msgBytes, err := json.Marshal(msgToDeliver)
				handleErr(err)
				err = amqpChannel.Publish(topicName, topicName+".confirmations", false, false, amqp2.Publishing{
					Headers: map[string]interface{}{
						"sagaId": info.Name(),
					},
					ContentType:     "application/json",
					Body:            msgBytes,
				})
				handleErr(err)
				handleErr(os.Remove(filePath))
				logger.Logf(log.InfoLevel, "SagaId: %s. Sent msg that account %s confirmed",info.Name(), uid)
			}
			handleErr(err)

		}
	}
}

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}


