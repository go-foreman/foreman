package amqp

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/stretchr/testify/require"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/go-foreman/foreman/testing/log"
	"github.com/golang/mock/gomock"
)

func TestChannel_Consume(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := log.NewNilLogger()
	channMock := NewMockAmqpChannel(ctrl)

	t.Run("second close() return an error", func(t *testing.T) {
		defer testLogger.Clear()

		ch := Channel{
			AmqpChannel: channMock,
			closed:      0,
			logger:      testLogger,
		}

		channMock.
			EXPECT().
			Close().
			Return(nil)

		assert.NoError(t, ch.Close())
		errSecondClose := ch.Close()
		assert.Error(t, errSecondClose)
		assert.Equal(t, errSecondClose, amqp.ErrClosed)
	})

	t.Run("successful retry", func(t *testing.T) {
		defer testLogger.Clear()

		ch := Channel{
			AmqpChannel: channMock,
			closed:      0,
			logger:      testLogger,
		}

		firstProducerCtx, cancelFirstProducer := context.WithCancel(context.Background())
		firstProducerCh := produceDeliveries(firstProducerCtx, "first", 1000)

		secondProducerCtx, cancelSecondProducer := context.WithCancel(context.Background())
		secondProducerCh := produceDeliveries(secondProducerCtx, "second", 1000)

		firstCall := channMock.
			EXPECT().
			Consume("q1", "q1", false, false, false, false, nil).
			Return(firstProducerCh, nil)

		secondCall := channMock.
			EXPECT().
			Consume("q1", "q1", false, false, false, false, nil).
			Return(nil, errors.New("error1 consuming")).
			After(firstCall)

		channMock.
			EXPECT().
			Consume("q1", "q1", false, false, false, false, nil).
			Do(func(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) {
				require.NoError(t, ch.Close())
				cancelSecondProducer()
			}).
			Return(secondProducerCh, nil).
			After(secondCall)

		channMock.
			EXPECT().
			Close().
			Return(nil)

		wg := &sync.WaitGroup{}
		wg.Add(1)

		go func() {
			time.Sleep(time.Millisecond * 100)
			cancelFirstProducer()
		}()

		go func() {
			defer wg.Done()

			d, err := ch.Consume("q1", "q1", false, false, false, false, nil)
			assert.NoError(t, err)

			opened := true
			for opened {
				_, opened = <-d
			}
		}()

		wg.Wait()

		testLogger.AssertContainsSubstr(t, "started consuming q1")
		testLogger.AssertContainsSubstr(t, "consume failed, err: error1 consuming")
		testLogger.AssertContainsSubstr(t, "retrying to reconnect consumer q1")
		testLogger.AssertContainsSubstr(t, "started consuming q1")
	})

	t.Run("reached retries limit", func(t *testing.T) {
		defer testLogger.Clear()

		ch := Channel{
			AmqpChannel:              channMock,
			closed:                   0,
			logger:                   testLogger,
			consumeReconnectionDelay: time.Millisecond * 100,
		}

		channMock.
			EXPECT().
			Consume("q1", "q1", false, false, false, false, nil).
			Return(nil, errors.New("error1 consuming")).
			Times(22)

		wg := &sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer wg.Done()

			d, err := ch.Consume("q1", "q1", false, false, false, false, nil)
			assert.NoError(t, err)

			opened := true
			for opened {
				_, opened = <-d
			}
		}()

		wg.Wait()

		testLogger.AssertContainsSubstr(t, fmt.Sprintf("Reached limit of reconnects %d", reconnectCount))
	})
}

func TestConnection_Channel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testLogger := log.NewNilLogger()
	underConnMock := NewMockUnderlyingConnection(ctrl)

	t.Run("create channel with an error", func(t *testing.T) {
		defer testLogger.Clear()

		conn := &Connection{
			logger:         testLogger,
			underlyingConn: underConnMock,
		}

		underConnMock.
			EXPECT().
			Channel().
			Return(nil, errors.New("some error"))

		ch, err := conn.Channel()
		assert.Error(t, err)
		assert.EqualError(t, err, "creating channel: some error")
		assert.Nil(t, ch)
	})

	// this test should be run by integration test, it's not possiblet to mock amqp.Channel :(
	//t.Run("recreate channel", func(t *testing.T) {
	//	defer testLogger.Clear()
	//
	//	conn := &Connection{
	//		logger:              testLogger,
	//		underlyingConn:      underConnMock,
	//		chReconnectionDelay: time.Millisecond * 200,
	//	}
	//
	//	amqpChannel := &amqp.Channel{}
	//
	//	firstCall := underConnMock.
	//		EXPECT().
	//		Channel().
	//		Return(amqpChannel, nil)
	//
	//	secondCall := underConnMock.
	//		EXPECT().
	//		Channel().
	//		Return(nil, errors.New("second call error")).
	//		After(firstCall)
	//
	//	newAmqpChannel := &amqp.Channel{}
	//
	//	underConnMock.
	//		EXPECT().
	//		Channel().
	//		Return(newAmqpChannel, nil).
	//		After(secondCall)
	//
	//	ch, err := conn.Channel()
	//	assert.NoError(t, err)
	//	assert.NotNil(t, ch)
	//
	//	// this sleep is needed here because notifyCloseCh will be initialized on goroutine.
	//	// needed only for test because there is no way to simulate NotifyClose using mock
	//	time.Sleep(time.Millisecond * 200)
	//
	//	assert.Panics(t, func() {
	//		ch.Close()
	//	})
	//
	//	time.Sleep(time.Second)
	//
	//	testLogger.AssertContainsSubstr(t, "created channel with connection")
	//	testLogger.AssertContainsSubstr(t, "channel closed, reason")
	//	testLogger.AssertContainsSubstr(t, "channel recreate failed, err: second call error")
	//	testLogger.AssertContainsSubstr(t, "channel recreation succeed")
	//})
}
