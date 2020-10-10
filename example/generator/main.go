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
	"time"
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
	for {
		time.Sleep(time.Second * 5)
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
			"messages_exchange",
			"messages_exchange.eventAndCommands",
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