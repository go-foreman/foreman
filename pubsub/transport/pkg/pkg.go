package pkg

import (
	"github.com/streadway/amqp"
	"time"
)

type IncomingPkg interface {
	ID() string
	Origin() string
	Payload() []byte
	Headers() map[string]interface{}
	Ack(options ...AcknowledgmentOption) error
	Nack(options ...AcknowledgmentOption) error
	Reject(options ...AcknowledgmentOption) error
	TraceId() string
	ReceivedAt() time.Time
	PublishedAt() time.Time
}

type ackOpts struct {
	multiple bool
	requeue  bool
}

type AcknowledgmentOption func(options *ackOpts)

func NewAmqpIncomingPackage(delivery amqp.Delivery, traceId, origin string) IncomingPkg {
	return &inAmqpPkg{traceId: traceId, origin: origin, receivedAt: time.Now(), delivery: delivery}
}

type inAmqpPkg struct {
	delivery   amqp.Delivery
	receivedAt time.Time
	origin     string
	attributes map[string]string
	traceId    string
}

func (i inAmqpPkg) ID() string {
	return i.delivery.MessageId
}

func (i inAmqpPkg) Origin() string {
	return i.origin
}

func (i inAmqpPkg) Payload() []byte {
	return i.delivery.Body
}

func (i inAmqpPkg) Headers() map[string]interface{} {
	return i.delivery.Headers
}

func (i inAmqpPkg) Attributes() map[string]string {
	return i.attributes
}

func (i inAmqpPkg) Ack(options ...AcknowledgmentOption) error {
	attrs := &ackOpts{}
	for _, opt := range options {
		opt(attrs)
	}

	return i.delivery.Ack(attrs.multiple)
}

func (i inAmqpPkg) Nack(options ...AcknowledgmentOption) error {
	ackOpts := &ackOpts{}
	for _, opt := range options {
		opt(ackOpts)
	}

	return i.delivery.Nack(ackOpts.multiple, ackOpts.requeue)
}

func (i inAmqpPkg) Reject(options ...AcknowledgmentOption) error {
	opts := &ackOpts{}
	for _, opt := range options {
		opt(opts)
	}

	return i.delivery.Reject(opts.requeue)
}

func (i inAmqpPkg) TraceId() string {
	return i.traceId
}

func (i inAmqpPkg) PublishedAt() time.Time {
	return i.delivery.Timestamp
}

func (i inAmqpPkg) ReceivedAt() time.Time {
	return i.receivedAt
}

type OutboundPkg interface {
	Payload() []byte
	ContentType() string
	Headers() map[string]interface{}
	Destination() DeliveryDestination
}

func NewOutboundPkg(payload []byte, contentType string, destination DeliveryDestination, headers map[string]interface{}) OutboundPkg {
	return &outboundPkg{payload: payload, contentType: contentType, destination: destination, headers: headers}
}

type outboundPkg struct {
	payload     []byte
	contentType string
	headers     map[string]interface{}
	destination DeliveryDestination
}

func (o outboundPkg) Payload() []byte {
	return o.payload
}

func (o outboundPkg) ContentType() string {
	return o.contentType
}

func (o outboundPkg) Headers() map[string]interface{} {
	return o.headers
}

func (o outboundPkg) Destination() DeliveryDestination {
	return o.destination
}

type DeliveryDestination struct {
	DestinationTopic string
	RoutingKey       string
}
