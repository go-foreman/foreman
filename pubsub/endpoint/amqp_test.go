package endpoint

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"

	"github.com/go-foreman/foreman/pubsub/transport"
	mockMessage "github.com/go-foreman/foreman/testing/mocks/pubsub/message"
	"github.com/stretchr/testify/assert"

	mockTransport "github.com/go-foreman/foreman/testing/mocks/pubsub/transport"
	"github.com/golang/mock/gomock"
)

func TestAmqpEndpoint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	marshallerTest := mockMessage.NewMockMarshaller(ctrl)
	transportTest := mockTransport.NewMockTransport(ctrl)

	destination := transport.DeliveryDestination{
		DestinationTopic: "messagebus_topic",
		RoutingKey:       "events",
	}

	amqpEndpoint := NewAmqpEndpoint(
		"amqp",
		transportTest,
		destination,
		marshallerTest,
	)

	t.Run("get name", func(t *testing.T) {
		assert.Equal(t, amqpEndpoint.Name(), "amqp")
	})

	t.Run("send", func(t *testing.T) {
		t.Run("error marshalling", func(t *testing.T) {
			payload := &testObj{}
			marshallerTest.
				EXPECT().
				Marshal(payload).
				Return(nil, errors.New("some error"))

			outcomingMsg := message.NewOutcomingMessage(payload)
			err := amqpEndpoint.Send(context.Background(), outcomingMsg)
			assert.Error(t, err)
			assert.EqualError(t, err, fmt.Sprintf("error serializing message %s to json : some error", outcomingMsg.UID()))
		})

		t.Run("sent", func(t *testing.T) {
			ctx := context.Background()
			payload := &testObj{}

			outcomingMsg := message.NewOutcomingMessage(payload, message.WithHeaders(message.Headers{"test": 1}))
			outboundPkg := transport.NewOutboundPkg([]byte("data"), "application/json", destination, outcomingMsg.Headers())

			t.Run("with error", func(t *testing.T) {
				marshallerTest.
					EXPECT().
					Marshal(payload).
					Return([]byte("data"), nil)

				transportTest.
					EXPECT().
					Send(ctx, outboundPkg).
					Return(errors.New("transport error"))

				err := amqpEndpoint.Send(ctx, outcomingMsg)
				assert.Error(t, err)
				assert.EqualError(t, err, "transport error")
			})

			t.Run("successfully sent", func(t *testing.T) {
				marshallerTest.
					EXPECT().
					Marshal(payload).
					Return([]byte("data"), nil)

				transportTest.
					EXPECT().
					Send(ctx, outboundPkg).
					Return(nil)
				assert.NoError(t, amqpEndpoint.Send(ctx, outcomingMsg))
			})

			t.Run("ctx canceled while waiting for the delay", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(ctx, time.Millisecond*200)
				defer cancel()

				marshallerTest.
					EXPECT().
					Marshal(payload).
					Return([]byte("data"), nil)

				err := amqpEndpoint.Send(ctx, outcomingMsg, WithDelay(time.Millisecond*300))
				assert.Error(t, err)
				assert.EqualError(t, err, fmt.Sprintf("failed to send message %s. Was waiting for the delay and parent ctx closed.", outcomingMsg.UID()))
			})

			t.Run("with delay opt", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(ctx, time.Millisecond*300)
				defer cancel()

				marshallerTest.
					EXPECT().
					Marshal(payload).
					Return([]byte("data"), nil)

				transportTest.
					EXPECT().
					Send(ctx, outboundPkg).
					Return(nil)

				err := amqpEndpoint.Send(ctx, outcomingMsg, WithDelay(time.Millisecond*200))
				assert.NoError(t, err)
			})

		})
	})

}
