package amqp

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

//go:generate mockgen --build_flags=--mod=mod -destination mock_test.go -package amqp . AmqpChannel,AmqpConnection,UnderlyingConnection

type AmqpChannel interface {
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	NotifyClose(c chan *amqp.Error) chan *amqp.Error
	Close() error
	Qos(prefetchCount, prefetchSize int, global bool) error
	Cancel(consumer string, noWait bool) error
}

type AmqpConnection interface {
	Channel() (AmqpChannel, error)
	Close() error
	IsClosed() bool
}

type UnderlyingConnection interface {
	Channel() (*amqp.Channel, error)
	Close() error
	IsClosed() bool
}
