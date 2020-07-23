package endpoint

import (
	"context"
	"encoding/json"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/transport/pkg"
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

func (a AmqpEndpoint) Send(ctx context.Context, message *message.Message, headers map[string]interface{}, userOpts ...DeliveryOption) error {
	opts := &deliveryOptions{}

	if userOpts != nil {
		for _, opt := range userOpts {
			if err := opt(opts); err != nil {
				return errors.Wrapf(err, "error compiling delivery options for message %s", message.ID)
			}
		}
	}

	dataToSend, err := json.Marshal(message)

	if err != nil {
		return errors.Wrapf(err, "error serializing message %s to json ", message.ID)
	}

	toSend := pkg.NewOutboundPkg(dataToSend, "application/json", a.destination, headers)

	if opts.delay != nil {
		waiter: for {
			select {
			case <- ctx.Done():
				return errors.Errorf("Failed to send message %s. Was waiting for the delay and parent ctx closed.", message.ID)
				case <- time.After(*opts.delay):
					break waiter //break for statement
			}
		}
	}

	return a.amqpTransport.Send(ctx, toSend)
}
