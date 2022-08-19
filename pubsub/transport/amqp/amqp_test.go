package amqp

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	transportMain "github.com/go-foreman/foreman/pubsub/transport"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/testing/log"
	"github.com/golang/mock/gomock"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestAmqpTransport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := log.NewNilLogger()
	connMock := NewMockAmqpConnection(ctrl)
	channMock := NewMockAmqpChannel(ctrl)

	t.Run("constructor", func(t *testing.T) {
		defer testLogger.Clear()

		underlyingConn := &amqp.Connection{}
		transport := NewTransport(underlyingConn, testLogger)
		assert.Nil(t, transport.Disconnect(context.Background()))
	})

	t.Run("connection is nil", func(t *testing.T) {
		transport := amqpTransport{
			connection: nil,
			logger:     testLogger,
		}

		err := transport.CreateTopic(context.Background(), &topicAnotherType{})
		assert.Error(t, err)
		assert.EqualError(t, err, "connection is nil")
	})

	t.Run("error creating publishing channel", func(t *testing.T) {
		transport := amqpTransport{
			connection: connMock,
			logger:     testLogger,
		}
		connMock.
			EXPECT().
			Channel().
			Return(nil, errors.New("chan err"))

		err := transport.CreateTopic(context.Background(), &topicAnotherType{})
		assert.Error(t, err)
		assert.EqualError(t, err, "creating publishing channel: chan err")
	})

	t.Run("create topic", func(t *testing.T) {
		defer testLogger.Clear()

		transport := amqpTransport{
			connection: connMock,
			logger:     testLogger,
		}

		connMock.
			EXPECT().
			Channel().
			Return(channMock, nil)
		channMock.
			EXPECT().
			ExchangeDeclare("someName", "topic", true, true, true, true, nil).
			Return(nil)

		err := transport.CreateTopic(context.Background(), Topic("someName", true, true, true, true))
		assert.NoError(t, err)

		t.Run("create topic with an error", func(t *testing.T) {
			channMock.
				EXPECT().
				ExchangeDeclare("someName", "topic", true, true, true, true, nil).
				Return(errors.New("some error"))
			err := transport.CreateTopic(context.Background(), Topic("someName", true, true, true, true))
			assert.Error(t, err)
			assert.EqualError(t, err, "some error")
		})

		t.Run("supplied topic of another type", func(t *testing.T) {
			err := transport.CreateTopic(context.Background(), &topicAnotherType{})
			assert.Error(t, err)
			assert.EqualError(t, err, "supplied topic is not an instance of amqp.Topic")
		})

		t.Run("check connection fails", func(t *testing.T) {
			transport := amqpTransport{
				connection: nil,
				logger:     testLogger,
			}
			err := transport.CreateTopic(context.Background(), &topicAnotherType{})
			assert.Error(t, err)
			assert.EqualError(t, err, "connection is nil")
		})
	})

	t.Run("create queue", func(t *testing.T) {
		defer testLogger.Clear()

		transport := amqpTransport{
			connection:        connMock,
			publishingChannel: channMock,
			logger:            testLogger,
		}

		channMock.
			EXPECT().
			QueueDeclare("queueName", true, true, true, true, nil).
			Return(amqp.Queue{}, nil)

		channMock.
			EXPECT().
			QueueBind("queueName", "binding1", "dest1", true, nil).
			Return(nil)
		channMock.
			EXPECT().
			QueueBind("queueName", "binding2", "dest2", false, nil).
			Return(nil)

		err := transport.CreateQueue(
			context.Background(),
			Queue("queueName", true, true, true, true),
			QueueBind("dest1", "binding1", true),
			QueueBind("dest2", "binding2", false),
		)
		assert.NoError(t, err)

		t.Run("create queue with an error", func(t *testing.T) {
			channMock.
				EXPECT().
				QueueDeclare("queueName", true, true, true, true, nil).
				Return(amqp.Queue{}, errors.New("some error queue"))

			err := transport.CreateQueue(
				context.Background(),
				Queue("queueName", true, true, true, true),
			)
			assert.Error(t, err)
			assert.EqualError(t, err, "some error queue")
		})

		t.Run("create queue with an error of bindings", func(t *testing.T) {
			channMock.
				EXPECT().
				QueueDeclare("queueName", true, true, true, true, nil).
				Return(amqp.Queue{}, nil)
			channMock.
				EXPECT().
				QueueBind("queueName", "binding1", "dest1", true, nil).
				Return(errors.New("binding error"))

			err := transport.CreateQueue(
				context.Background(),
				Queue("queueName", true, true, true, true),
				QueueBind("dest1", "binding1", true),
			)
			assert.Error(t, err)
			assert.EqualError(t, err, "binding error")
		})

		t.Run("supplied queue of another type", func(t *testing.T) {
			err := transport.CreateQueue(context.Background(), &queueAnotherType{})
			assert.Error(t, err)
			assert.EqualError(t, err, "supplied Queue is not an instance of amqp.amqpQueue")
		})

		t.Run("supplied queue bind of another type", func(t *testing.T) {
			err := transport.CreateQueue(context.Background(), Queue("queueName", true, true, true, true), &queueBindAnotherType{})
			assert.Error(t, err)
			assert.EqualError(t, err, "one of supplied QueueBinds is not an instance of amqp.amqpQueueBind")
		})

		t.Run("check connection fails", func(t *testing.T) {
			transport := amqpTransport{
				connection: nil,
				logger:     testLogger,
			}
			err := transport.CreateQueue(context.Background(), &queueAnotherType{})
			assert.Error(t, err)
			assert.EqualError(t, err, "connection is nil")
		})
	})

	t.Run("create queue with explicit classic type", func(t *testing.T) {
		defer testLogger.Clear()

		transport := amqpTransport{
			connection:        connMock,
			publishingChannel: channMock,
			logger:            testLogger,
		}

		channMock.
			EXPECT().
			QueueDeclare("queueName", true, true, true, true, amqp.Table{
				"x-queue-type": "classic",
			}).
			Return(amqp.Queue{}, nil)

		channMock.
			EXPECT().
			QueueBind("queueName", "binding1", "dest1", true, nil).
			Return(nil)
		channMock.
			EXPECT().
			QueueBind("queueName", "binding2", "dest2", false, nil).
			Return(nil)

		err := transport.CreateQueue(
			context.Background(),
			Queue("queueName", true, true, true, true, WithQueueType(QueueTypeClassic)),
			QueueBind("dest1", "binding1", true),
			QueueBind("dest2", "binding2", false),
		)
		assert.NoError(t, err)
	})

	t.Run("create queue with explicit quorum type", func(t *testing.T) {
		defer testLogger.Clear()

		transport := amqpTransport{
			connection:        connMock,
			publishingChannel: channMock,
			logger:            testLogger,
		}

		channMock.
			EXPECT().
			QueueDeclare("queueName", true, true, true, true, amqp.Table{
				"x-queue-type": "quorum",
			}).
			Return(amqp.Queue{}, nil)

		channMock.
			EXPECT().
			QueueBind("queueName", "binding1", "dest1", true, nil).
			Return(nil)
		channMock.
			EXPECT().
			QueueBind("queueName", "binding2", "dest2", false, nil).
			Return(nil)

		err := transport.CreateQueue(
			context.Background(),
			Queue("queueName", true, true, true, true, WithQueueType(QueueTypeQuorum)),
			QueueBind("dest1", "binding1", true),
			QueueBind("dest2", "binding2", false),
		)
		assert.NoError(t, err)
	})

	t.Run("send", func(t *testing.T) {
		defer testLogger.Clear()

		transport := amqpTransport{
			connection:        connMock,
			publishingChannel: channMock,
			logger:            testLogger,
		}

		outboundPkg := transportMain.NewOutboundPkg(
			[]byte("data"),
			"application/json",
			transportMain.DeliveryDestination{
				DestinationTopic: "someTopic",
				RoutingKey:       "someKey",
			},
			map[string]interface{}{
				"key": "val",
			},
		)

		channMock.
			EXPECT().
			Publish("someTopic", "someKey", false, false, amqp.Publishing{
				Headers:     outboundPkg.Headers(),
				ContentType: outboundPkg.ContentType(),
				Body:        outboundPkg.Payload(),
			}).
			Return(nil)

		err := transport.Send(context.Background(), outboundPkg)
		assert.NoError(t, err)

		t.Run("publish with an error", func(t *testing.T) {
			channMock.
				EXPECT().
				Publish("someTopic", "someKey", false, false, amqp.Publishing{
					Headers:     outboundPkg.Headers(),
					ContentType: outboundPkg.ContentType(),
					Body:        outboundPkg.Payload(),
				}).
				Return(errors.New("publish error"))

			err := transport.Send(context.Background(), outboundPkg)
			assert.Error(t, err)
			assert.EqualError(t, err, "sending out pkg: publish error")
		})

		t.Run("check connection fails", func(t *testing.T) {
			transport := amqpTransport{
				connection: nil,
				logger:     testLogger,
			}
			err := transport.Send(context.Background(), outboundPkg)
			assert.Error(t, err)
			assert.EqualError(t, err, "connection is nil")
		})

		t.Run("with options", func(t *testing.T) {
			channMock.
				EXPECT().
				Publish("someTopic", "someKey", true, true, amqp.Publishing{
					Headers:     outboundPkg.Headers(),
					ContentType: outboundPkg.ContentType(),
					Body:        outboundPkg.Payload(),
				}).
				Return(nil)

			err := transport.Send(context.Background(), outboundPkg, WithMandatory(), WithImmediate())
			assert.NoError(t, err)
		})

		t.Run("with wrong options", func(t *testing.T) {

		})
	})

	t.Run("disconnect", func(t *testing.T) {
		t.Run("no connection or pub channel", func(t *testing.T) {
			transport := amqpTransport{
				connection: nil,
				logger:     testLogger,
			}

			err := transport.Disconnect(context.Background())
			assert.NoError(t, err)
		})

		t.Run("error closing pub channel", func(t *testing.T) {
			transport := amqpTransport{
				connection:        connMock,
				publishingChannel: channMock,
				logger:            testLogger,
			}
			channMock.
				EXPECT().
				Close().
				Return(errors.New("err closing pub channel"))

			err := transport.Disconnect(context.Background())
			assert.Error(t, err)
			assert.EqualError(t, err, "closing publishing channel: err closing pub channel")
		})

		t.Run("err closing one of consuming channels", func(t *testing.T) {
			additionalChMock := NewMockAmqpChannel(ctrl)

			transport := amqpTransport{
				connection:        connMock,
				publishingChannel: channMock,
				mutex:             &sync.Mutex{},
				consumingChannels: map[AmqpChannel]struct{}{
					additionalChMock: {},
				},
				logger: testLogger,
			}

			channMock.
				EXPECT().
				Close().
				Return(nil)

			additionalChMock.
				EXPECT().
				Close().
				Return(errors.New("some err"))

			err := transport.Disconnect(context.Background())
			assert.Error(t, err)
			assert.EqualError(t, err, "closing one of consuming channels: some err")
		})

		t.Run("successfully closed all channels", func(t *testing.T) {
			additionalChMock := NewMockAmqpChannel(ctrl)

			transport := amqpTransport{
				connection:        connMock,
				publishingChannel: channMock,
				mutex:             &sync.Mutex{},
				consumingChannels: map[AmqpChannel]struct{}{
					additionalChMock: {},
				},
				logger: testLogger,
			}

			channMock.
				EXPECT().
				Close().
				Return(nil)

			additionalChMock.
				EXPECT().
				Close().
				Return(nil)

			err := transport.Disconnect(context.Background())
			assert.NoError(t, err)
		})
	})

	t.Run("consume", func(t *testing.T) {
		t.Run("successfully consume two queues", func(t *testing.T) {
			defer testLogger.Clear()

			transport := amqpTransport{
				connection:        connMock,
				mutex:             &sync.Mutex{},
				consumingChannels: map[AmqpChannel]struct{}{},
				logger:            testLogger,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			q1 := Queue("q1", true, true, true, true)
			q2 := Queue("q2", true, true, true, true)

			ctxQ1, cancelQ1 := context.WithCancel(context.Background())

			ctxQ2, cancelQ2 := context.WithCancel(context.Background())

			q1Chan := produceDeliveries(ctxQ1, "q1", 100)
			q2Chan := produceDeliveries(ctxQ2, "q2", 100)

			connMock.
				EXPECT().
				Channel().
				Return(channMock, nil).Times(2)

			channMock.
				EXPECT().
				Qos(100, 0, false)

			channMock.
				EXPECT().
				Consume(q1.Name(), q1.Name(), false, true, true, true, nil).
				Return(q1Chan, nil)
			channMock.
				EXPECT().
				Consume(q2.Name(), q2.Name(), false, true, true, true, nil).
				Return(q2Chan, nil)

			channMock.
				EXPECT().
				Cancel(q1.Name(), false).
				Return(nil)
			channMock.
				EXPECT().
				Cancel(q2.Name(), false).
				Return(nil)
			channMock.
				EXPECT().
				Close().
				Return(nil)

			packagesChan, err := transport.Consume(
				ctx,
				[]transportMain.Queue{q1, q2},
				WithNoLocal(),
				WithQosPrefetchCount(100),
				WithExclusive(),
				WithNoWait(),
			)

			assert.NoError(t, err)

			go func() {
				time.Sleep(time.Second * 2)
				cancelQ1()
				cancelQ2()
			}()

			var receivedPackages []transportMain.IncomingPkg

			firstPackageDone := false
			for pkg := range packagesChan {
				if !firstPackageDone {
					transport.mutex.Lock()
					// once all consume goroutines started need to verify that consuming channel is recorded in the map. Need to verify just once, but at each package is also fine.
					assert.Contains(t, transport.consumingChannels, channMock)
					transport.mutex.Unlock()
					firstPackageDone = true
				}
				receivedPackages = append(receivedPackages, pkg)
			}

			assert.Len(t, receivedPackages, 200)

			testLogger.AssertContainsSubstr(t, fmt.Sprintf("amqp consumer closed channel for queue %s", q1.Name()))
			testLogger.AssertContainsSubstr(t, fmt.Sprintf("amqp consumer closed channel for queue %s", q2.Name()))

			testLogger.AssertContainsSubstr(t, fmt.Sprintf("canceling consumer %s", q1.Name()))
			testLogger.AssertContainsSubstr(t, fmt.Sprintf("canceling consumer %s", q2.Name()))

			testLogger.AssertContainsSubstr(t, fmt.Sprintf("canceled consumer %s", q1.Name()))
			testLogger.AssertContainsSubstr(t, fmt.Sprintf("canceled consumer %s", q2.Name()))

			testLogger.AssertContainsSubstr(t, "closed consumer channel")

			transport.mutex.Lock()
			assert.Empty(t, transport.consumingChannels)
			transport.mutex.Unlock()
		})

		t.Run("one of consumers failed", func(t *testing.T) {
			defer testLogger.Clear()

			transport := amqpTransport{
				connection:        connMock,
				mutex:             &sync.Mutex{},
				consumingChannels: map[AmqpChannel]struct{}{},
				logger:            testLogger,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			q1 := Queue("q1", true, true, true, true)
			q2 := Queue("q2", true, true, true, true)

			ctxQ1, cancelQ1 := context.WithCancel(context.Background())
			defer cancelQ1()

			q1Chan := produceDeliveries(ctxQ1, "q1", 100)

			connMock.
				EXPECT().
				Channel().
				Return(channMock, nil).Times(2)

			channMock.
				EXPECT().
				Consume(q1.Name(), q1.Name(), false, false, false, false, nil).
				Return(q1Chan, nil)
			channMock.
				EXPECT().
				Consume(q2.Name(), q2.Name(), false, false, false, false, nil).
				Return(nil, errors.New("some error on the second consumer"))

			channMock.
				EXPECT().
				Cancel(q1.Name(), false).
				Return(errors.New("error Cancel(q1)"))
			channMock.
				EXPECT().
				Close().
				Return(errors.New("error Close()"))

			packagesChan, err := transport.Consume(
				ctx,
				[]transportMain.Queue{q1, q2},
			)

			assert.Error(t, err)
			assert.EqualError(t, err, "consuming q2: some error on the second consumer")
			assert.Nil(t, packagesChan)

			// waiting for goroutines to finish
			time.Sleep(time.Millisecond * 200)

			testLogger.AssertContainsSubstr(t, fmt.Sprintf("canceled context. Stopped consuming queue %s", q1.Name()))
			testLogger.AssertContainsSubstr(t, fmt.Sprintf("canceling consumer %s", q1.Name()))
			testLogger.AssertContainsSubstr(t, fmt.Sprintf("error canceling consumer %s. error Cancel(q1)", q1.Name()))
			testLogger.AssertContainsSubstr(t, "error closing amqp channel. error Close()")
		})

		t.Run("error creating consuming channel", func(t *testing.T) {
			defer testLogger.Clear()

			transport := amqpTransport{
				connection:        connMock,
				publishingChannel: channMock,
				mutex:             &sync.Mutex{},
				consumingChannels: map[AmqpChannel]struct{}{},
				logger:            testLogger,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			q1 := Queue("q1", true, true, true, true)

			connMock.
				EXPECT().
				Channel().
				Return(nil, errors.New("some error"))

			packagesChan, err := transport.Consume(
				ctx,
				[]transportMain.Queue{q1},
			)

			assert.Error(t, err)
			assert.EqualError(t, err, "some error")
			assert.Nil(t, packagesChan)
		})
	})

}

type topicAnotherType struct {
}

func (t topicAnotherType) Name() string {
	return "xxx"
}

type queueAnotherType struct {
}

func (q queueAnotherType) Name() string {
	return "xxx"
}

type queueBindAnotherType struct {
}

func (q queueBindAnotherType) DestinationTopic() string {
	return "xxx"
}

func (q queueBindAnotherType) BindingKey() string {
	return "xxx"
}

func produceDeliveries(ctx context.Context, prefix string, count int) <-chan amqp.Delivery {
	d := make(chan amqp.Delivery)
	go func() {
		defer close(d)
		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
				return
			case d <- amqp.Delivery{Body: []byte(fmt.Sprintf("%s-%d", prefix, i))}:

			}
		}
	}()

	return d
}
