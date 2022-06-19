package subscriber

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/go-foreman/foreman/pubsub/transport/amqp"

	"github.com/go-foreman/foreman/testing/log"
	subscriberMock "github.com/go-foreman/foreman/testing/mocks/pubsub/subscriber"

	transportMock "github.com/go-foreman/foreman/testing/mocks/pubsub/transport"
	"github.com/golang/mock/gomock"
)

func TestSubscriber(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testTransport := transportMock.NewMockTransport(ctrl)
	testProcessor := subscriberMock.NewMockProcessor(ctrl)

	testLogger := log.NewNilLogger()

	t.Run("error consume", func(t *testing.T) {
		defer testLogger.Clear()

		ctx := context.Background()
		queues := []transport.Queue{
			amqp.Queue("first", false, false, false, false),
		}
		testTransport.
			EXPECT().
			Consume(gomock.Any(), queues).
			Return(nil, errors.New("consume err"))

		subscriber := NewSubscriber(testTransport, testProcessor, testLogger)
		err := subscriber.Run(ctx, queues...)
		assert.Error(t, err)
		assert.EqualError(t, err, "consume err")

	})

	t.Run("process packages and exit by cancelling the ctx", func(t *testing.T) {
		defer testLogger.Clear()

		queues := []transport.Queue{
			amqp.Queue("second", false, false, false, false),
		}
		subscriber := NewSubscriber(testTransport, testProcessor, testLogger)
		ctx, cancel := context.WithCancel(context.Background())

		doneCh := make(chan struct{})

		pkgsChan := producePackages(ctrl, testProcessor, 1000, doneCh)

		testTransport.
			EXPECT().
			Consume(gomock.AssignableToTypeOf(ctx), queues).
			Return(pkgsChan, nil)

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := subscriber.Run(ctx, queues...); err != nil {
				assert.NoError(t, err)
			}
		}()

		<-doneCh

		cancel()

		wg.Wait()

		assert.Len(t, pkgsChan, 0)
		close(pkgsChan)

		assert.Contains(t, testLogger.Messages(), "Subscriber's context was canceled")
	})

}

func producePackages(ctrl *gomock.Controller, processorMock *subscriberMock.MockProcessor, count int, done chan struct{}) chan transport.IncomingPkg {
	respChan := make(chan transport.IncomingPkg)

	go func() {
		defer func() {
			done <- struct{}{}
		}()
		for i := 0; i < count; i++ {
			inPkg := transportMock.NewMockIncomingPkg(ctrl)
			inPkg.EXPECT().UID().Return(fmt.Sprintf("%d", i)).Times(2)
			inPkg.EXPECT().Ack().Return(nil)
			processorMock.
				EXPECT().
				Process(gomock.Any(), inPkg).
				Return(nil)
			respChan <- inPkg
		}
	}()

	return respChan
}
