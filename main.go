package main

import "github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/plugins/amqp"

func main() {
	amqpTransport := amqp.NewTransport("amqp://admin:admin123@127.0.0.1:5672")
}
