package transport

import (
	"time"
)

type IncomingPkg interface {
	UID() string
	Origin() string
	Payload() []byte
	Headers() map[string]interface{}
	Ack(options ...AcknowledgmentOption) error
	Nack(options ...AcknowledgmentOption) error
	Reject(options ...AcknowledgmentOption) error
	ReceivedAt() time.Time
	PublishedAt() time.Time
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

type AcknowledgmentOption func(options map[string]interface{})
