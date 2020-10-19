package endpoint

import (
	"context"
	"encoding/json"
	"github.com/go-foreman/foreman/pkg/pubsub/message"
	"github.com/go-foreman/foreman/pkg/pubsub/transport"
	"github.com/go-foreman/foreman/pkg/pubsub/transport/pkg"
	"github.com/pkg/errors"
	"time"
)

type AmqpEndpoint struct {
	amqpTransport transport.Transport
	destination   pkg.DeliveryDestination
	name          string
}

func NewAmqpEndpoint(name string, amqpTransport transport.Transport, destination pkg.DeliveryDestination) Endpoint {
	return &AmqpEndpoint{name: name, amqpTransport: amqpTransport, destination: destination}
}

func (a AmqpEndpoint) Name() string {
	return a.name
}

func (a AmqpEndpoint) Send(ctx context.Context, message *message.Message, headers map[string]interface{}, opts ...DeliveryOption) error {
	deliveryOpts := &deliveryOptions{}

	if opts != nil {
		for _, opt := range opts {
			if err := opt(deliveryOpts); err != nil {
				return errors.Wrapf(err, "error compiling delivery options for message %s", message.ID)
			}
		}
	}

	dataToSend, err := json.Marshal(message)

	if err != nil {
		return errors.Wrapf(err, "error serializing message %s to json ", message.ID)
	}

	toSend := pkg.NewOutboundPkg(dataToSend, "application/json", a.destination, headers)

	if deliveryOpts.delay != nil {
	waiter:
		for {
			select {
			case <-ctx.Done():
				return errors.Errorf("Failed to send message %s. Was waiting for the delay and parent ctx closed.", message.ID)
			case <-time.After(*deliveryOpts.delay):
				break waiter //break for statement
			}
		}
	}

	return a.amqpTransport.Send(ctx, toSend)
}
