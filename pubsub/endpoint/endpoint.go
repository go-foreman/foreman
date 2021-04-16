package endpoint

import (
	"context"
	"github.com/go-foreman/foreman/pubsub/message"
	"time"
)

type Endpoint interface {
	Name() string
	Send(ctx context.Context, message *message.OutcomingMessage, options ...DeliveryOption) error
}

type deliveryOptions struct {
	delay *time.Duration
	headers message.Headers
}

func WithDelay(delay time.Duration) DeliveryOption {
	return func(o *deliveryOptions) error {
		o.delay = &delay
		return nil
	}
}

type DeliveryOption func(o *deliveryOptions) error
