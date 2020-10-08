package endpoint

import (
	"context"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"time"
)

type Endpoint interface {
	Name() string
	Send(ctx context.Context, message *message.Message, headers map[string]interface{}, options ...DeliveryOption) error
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
