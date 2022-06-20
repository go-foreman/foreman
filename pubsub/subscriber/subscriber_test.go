package subscriber

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

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

	t.Run("worker was waiting for a job to start and returned back to the pool", func(t *testing.T) {
		defer testLogger.Clear()

		ctx, cancel := context.WithCancel(context.Background())
		queues := []transport.Queue{
			amqp.Queue("first", false, false, false, false),
		}

		respChan := make(chan transport.IncomingPkg)

		testTransport.
			EXPECT().
			Consume(gomock.Any(), queues).
			Return(respChan, nil)

		config := &Config{
			WorkersCount:                   10,
			WorkerWaitingAssignmentTimeout: time.Second * 2,
			PackageProcessingMaxTime:       time.Second * 10,
			GracefulShutdownTimeout:        time.Second * 10,
		}

		subscriber := NewSubscriber(testTransport, testProcessor, testLogger, WithConfig(config))

		go func() {
			err := subscriber.Run(ctx, queues...)
			assert.NoError(t, err)
		}()

		//wait for package to process
		time.Sleep(time.Second * 3)

		cancel()

		assert.Contains(t, testLogger.Messages(), fmt.Sprintf("worker was waiting %s for a job to start. returning him to the pool", config.WorkerWaitingAssignmentTimeout.String()))

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

	t.Run("graceful shutdown timeout", func(t *testing.T) {
		defer testLogger.Clear()

		defer func() {
			for _, l := range testLogger.Messages() {
				t.Log(l)
			}
		}()

		queues := []transport.Queue{
			amqp.Queue("second", false, false, false, false),
		}
		subscriber := NewSubscriber(testTransport, testProcessor, testLogger, WithConfig(&Config{
			WorkersCount:                   10,
			WorkerWaitingAssignmentTimeout: time.Second * 3,
			PackageProcessingMaxTime:       time.Second * 10,
			GracefulShutdownTimeout:        time.Second * 2,
		}))
		ctx, cancel := context.WithCancel(context.Background())

		pkgsChan := make(chan transport.IncomingPkg, 1)

		testTransport.
			EXPECT().
			Consume(gomock.AssignableToTypeOf(ctx), queues).
			Return(pkgsChan, nil)

		inPkg := transportMock.NewMockIncomingPkg(ctrl)
		inPkg.EXPECT().UID().Return("111").Times(2)
		inPkg.EXPECT().Ack().Return(nil)

		startedProcessingNotifier := make(chan struct{})

		testProcessor.
			EXPECT().
			Process(gomock.Any(), inPkg).
			Do(func(ctx context.Context, inPkg transport.IncomingPkg) {
				startedProcessingNotifier <- struct{}{}
				time.Sleep(time.Second * 3)
			}).
			Return(nil)

		pkgsChan <- inPkg

		go func() {
			// wait for the package to be processed
			<-startedProcessingNotifier
			// trigger gracefulShutdown
			cancel()
		}()

		if err := subscriber.Run(ctx, queues...); err != nil {
			assert.NoError(t, err)
		}

		//exiting here without this sleep will stop all goroutines and processed package will abort it's execution.
		time.Sleep(time.Second * 2)

		assert.Contains(t, testLogger.Messages(), "Stopped gracefulShutdown because of canceled parent ctx")
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
