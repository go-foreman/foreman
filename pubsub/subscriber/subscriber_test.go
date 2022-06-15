package subscriber

import (
	"context"
	"fmt"
	"testing"
	"time"

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

	t.Run("process packages and exit by cancelling the ctx", func(t *testing.T) {
		queues := []transport.Queue{
			amqp.Queue("first", false, false, false, false),
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

		go func() {
			if err := subscriber.Run(ctx, queues...); err != nil {
				assert.NoError(t, err)
			}
		}()

		<-doneCh

		cancel()

		time.Sleep(time.Second)

		assert.Len(t, pkgsChan, 0)
		close(pkgsChan)
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
