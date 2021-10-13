package endpoint

import (
	"context"
	"time"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/pkg/errors"
)

type AmqpEndpoint struct {
	amqpTransport transport.Transport
	destination   transport.DeliveryDestination
	msgMarshaller message.Marshaller
	name          string
}

func NewAmqpEndpoint(name string, amqpTransport transport.Transport, destination transport.DeliveryDestination, msgMarshaller message.Marshaller) Endpoint {
	return &AmqpEndpoint{name: name, amqpTransport: amqpTransport, destination: destination, msgMarshaller: msgMarshaller}
}

func (a AmqpEndpoint) Name() string {
	return a.name
}

func (a AmqpEndpoint) Send(ctx context.Context, msg *message.OutcomingMessage, opts ...DeliveryOption) error {
	deliveryOpts := &deliveryOptions{}

	for _, opt := range opts {
		if err := opt(deliveryOpts); err != nil {
			return errors.Wrapf(err, "error compiling delivery options for message %s", msg.UID())
		}
	}

	dataToSend, err := a.msgMarshaller.Marshal(msg.Payload())

	if err != nil {
		return errors.Wrapf(err, "error serializing message %s to json ", msg.UID())
	}

	toSend := transport.NewOutboundPkg(dataToSend, "application/json", a.destination, msg.Headers())

	if deliveryOpts.delay != nil {
	waiter:
		for {
			select {
			case <-ctx.Done():
				return errors.Errorf("Failed to send message %s. Was waiting for the delay and parent ctx closed.", msg.UID())
			case <-time.After(*deliveryOpts.delay):
				break waiter //break for statement
			}
		}
	}

	return a.amqpTransport.Send(ctx, toSend)
}
