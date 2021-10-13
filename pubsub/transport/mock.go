package transport

import (
	"context"
	"time"
)

func NewStubTransport() *stubTransport {
	return &stubTransport{
		topics:       map[string]chan IncomingPkg{},
		queueBinding: map[string][]string{},
	}
}

type stubTransport struct {
	topics       map[string]chan IncomingPkg
	queueBinding map[string][]string
}

func (m *stubTransport) CreateTopic(ctx context.Context, topic Topic) error {
	m.topics[topic.Name()] = make(chan IncomingPkg)
	return nil
}

func (m *stubTransport) CreateQueue(ctx context.Context, queue Queue, queueBind ...QueueBind) error {
	topics, exist := m.queueBinding[queue.Name()]
	if !exist {
		topics = make([]string, 0)
	}

	for _, qb := range queueBind {
		topics = append(topics, qb.DestinationTopic())
	}

	m.queueBinding[queue.Name()] = topics

	return nil
}

func (m *stubTransport) Consume(ctx context.Context, queues []Queue, options ...ConsumeOpts) (<-chan IncomingPkg, error) {
	income := make(chan IncomingPkg)

	for i := range queues {
		go func(q Queue) {
			for _, topic := range m.queueBinding[q.Name()] {
				msgs := m.topics[topic]
				for {
					select {
					case msg, open := <-msgs:
						if !open {
							return
						}

						income <- msg
					case <-ctx.Done():
						return
					}
				}
			}
		}(queues[i])
	}

	return income, nil
}

func (m *stubTransport) Send(ctx context.Context, outboundPkg OutboundPkg, options ...SendOpts) error {
	ch := m.topics[outboundPkg.Destination().DestinationTopic]
	inc := inPkg{
		payload:     outboundPkg.Payload(),
		contentType: outboundPkg.ContentType(),
		headers:     outboundPkg.Headers(),
		origin:      "origin",
		attributes:  make(map[string]string),
		traceId:     "traceId",
	}

	ch <- inc
	return nil
}

func (m *stubTransport) Connect(context.Context) error {
	return nil
}

func (m *stubTransport) Disconnect(context.Context) error {
	return nil
}

type inPkg struct {
	payload     []byte
	contentType string
	headers     map[string]interface{}
	origin      string
	attributes  map[string]string
	traceId     string
}

func (i inPkg) UID() string {
	return "id"
}

func (i inPkg) Origin() string {
	return i.origin
}

func (i inPkg) Payload() []byte {
	return i.payload
}

func (i inPkg) Headers() map[string]interface{} {
	return i.headers
}

func (i inPkg) Attributes() map[string]string {
	return i.attributes
}

func (i inPkg) Ack(options ...AcknowledgmentOption) error {
	return nil
}

func (i inPkg) Nack(options ...AcknowledgmentOption) error {
	return nil
}

func (i inPkg) Reject(options ...AcknowledgmentOption) error {
	return nil
}

func (i inPkg) PublishedAt() time.Time {
	return time.Now()
}

func (i inPkg) ReceivedAt() time.Time {
	return time.Now()
}
