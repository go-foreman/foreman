package amqp

import (
	"time"

	"github.com/go-foreman/foreman/pubsub/transport"
	amqp "github.com/rabbitmq/amqp091-go"
)

type delivery struct {
	msg *amqp.Delivery
}

func (d delivery) Reject(requeue bool) error {
	return d.msg.Reject(requeue)
}

func (d delivery) Nack(multiple, requeue bool) error {
	return d.msg.Nack(multiple, requeue)
}

func (d delivery) Ack(multiple bool) error {
	return d.msg.Ack(multiple)
}

func (d delivery) Timestamp() time.Time {
	return d.msg.Timestamp
}

func (d delivery) Headers() amqp.Table {
	return d.msg.Headers
}

func (d delivery) Body() []byte {
	return d.msg.Body
}

type inAmqpPkg struct {
	delivery   Delivery
	receivedAt time.Time
	origin     string
	attributes map[string]string
}

func (i inAmqpPkg) UID() string {
	uidVal, ok := i.delivery.Headers()["uid"]
	if ok {
		return uidVal.(string)
	}
	return ""
}

func (i inAmqpPkg) Origin() string {
	return i.origin
}

func (i inAmqpPkg) Payload() []byte {
	return i.delivery.Body()
}

func (i inAmqpPkg) Headers() map[string]interface{} {
	if i.delivery.Headers() == nil {
		return make(amqp.Table)
	}

	return i.delivery.Headers()
}

func (i inAmqpPkg) Attributes() map[string]string {
	return i.attributes
}

func (i inAmqpPkg) Ack(options ...transport.AcknowledgmentOption) error {
	ackOpts := collectOpts(options...)

	return i.delivery.Ack(ackOpts.multiple)
}

func (i inAmqpPkg) Nack(options ...transport.AcknowledgmentOption) error {
	ackOpts := collectOpts(options...)

	return i.delivery.Nack(ackOpts.multiple, ackOpts.requeue)
}

func (i inAmqpPkg) Reject(options ...transport.AcknowledgmentOption) error {
	ackOpts := collectOpts(options...)

	return i.delivery.Reject(ackOpts.requeue)
}

func (i inAmqpPkg) PublishedAt() time.Time {
	return i.delivery.Timestamp()
}

func (i inAmqpPkg) ReceivedAt() time.Time {
	return i.receivedAt
}

func WithRequeue() transport.AcknowledgmentOption {
	return func(options map[string]interface{}) {
		options["requeue"] = true
	}
}

func WithMultiple() transport.AcknowledgmentOption {
	return func(options map[string]interface{}) {
		options["multiple"] = true
	}
}

type ackOpts struct {
	requeue  bool
	multiple bool
}

func collectOpts(passedOpts ...transport.AcknowledgmentOption) *ackOpts {
	optsMap := map[string]interface{}{}
	for _, opt := range passedOpts {
		opt(optsMap)
	}

	return mapToOpts(optsMap)
}

func mapToOpts(passedOpts map[string]interface{}) *ackOpts {
	opts := &ackOpts{}

	if passedOpts != nil {
		if requeueVal, exists := passedOpts["requeue"]; exists {
			if requeue, isBool := requeueVal.(bool); isBool {
				opts.requeue = requeue
			}
		}

		if multipleVal, exists := passedOpts["multiple"]; exists {
			if multiple, isBool := multipleVal.(bool); isBool {
				opts.multiple = multiple
			}
		}
	}

	return opts
}
