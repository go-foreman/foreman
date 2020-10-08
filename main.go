package main

import (
	"bytes"
	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/plugins/amqp"
	"log"
)

func main() {
	var buf bytes.Buffer
	logger := log.New(&buf, "logger: ", log.Lshortfile)
	amqpTransport := amqp.NewTransport("amqp://admin:admin123@127.0.0.1:5672", logger)
	amqpTransport.Connect(context.Background())
}
