package endpoint

import (
	"context"
	"time"

	"github.com/go-foreman/foreman/pubsub/message"
)

type Endpoint interface {
	// Name is a unique name of the endpoint
	Name() string
	// Send sends a message to specified implementation
	Send(ctx context.Context, message *message.OutcomingMessage, options ...DeliveryOption) error
}

type deliveryOptions struct {
	delay *time.Duration
}

func WithDelay(delay time.Duration) DeliveryOption {
	return func(o *deliveryOptions) error {
		o.delay = &delay
		return nil
	}
}

type DeliveryOption func(o *deliveryOptions) error
