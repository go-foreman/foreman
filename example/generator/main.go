package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase/account"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	sagaContracts "github.com/kopaygorodsky/brigadier/pkg/saga/contracts"
	streadwayAmqp "github.com/streadway/amqp"
)

func main() {
	conn, err := streadwayAmqp.Dial("amqp://admin:admin123@127.0.0.1:5672")
	if err != nil {
		panic(err)
	}

	ch, err := conn.Channel()

	if err != nil {
		panic(err)
	}
	for i := 0; i < 100000; i++ {
		uid := uuid.New().String()
		registerAccountSaga := &account.RegisterAccountSaga{
			UID:          uid,
			Email:        fmt.Sprintf("account-%s@github.com", uid),
			RetriesLimit: 10,
		}
		startSagaCmd := &sagaContracts.StartSagaCommand{
			SagaId:   uuid.New().String(),
			SagaName: scheme.WithStruct(registerAccountSaga)(),
			Saga:     registerAccountSaga,
		}
		messageToDeliver := message.NewCommandMessage(startSagaCmd)
		msgBytes, err := json.Marshal(messageToDeliver)
		if err != nil {
			panic(err)
		}

		err = ch.Publish(
			"messagebus_exchange",
			"messagebus_exchange.eventAndCommands",
			false,
			false,
			streadwayAmqp.Publishing{
				ContentType: "application/json",
				Body:        msgBytes,
			},
		)
		if err != nil {
			panic(err)
		}
	}
}
