package endpoint

import (
	"context"
	"time"

	"github.com/go-foreman/foreman/pubsub/message"
)

// Endpoint knows where to deliver a message
type Endpoint interface {
	// Name is a unique name of the endpoint
	Name() string
	// Send sends a message with specified implementation
	Send(ctx context.Context, message *message.OutcomingMessage, options ...DeliveryOption) error
}

type deliveryOptions struct {
	delay *time.Duration
}

// WithDelay option waits specified duration before delivering a message
func WithDelay(delay time.Duration) DeliveryOption {
	return func(o *deliveryOptions) error {
		o.delay = &delay
		return nil
	}
}

type DeliveryOption func(o *deliveryOptions) error
