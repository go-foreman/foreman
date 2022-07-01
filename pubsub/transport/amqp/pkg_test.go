package amqp

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestPkgDelivery(t *testing.T) {
	timeNow := time.Now()

	d := &delivery{msg: &amqp.Delivery{
		Headers:   nil,
		Timestamp: timeNow,
		Body:      []byte("payload"),
	}}

	var headers amqp.Table

	assert.Equal(t, timeNow, d.Timestamp())
	assert.Equal(t, []byte("payload"), d.Body())
	assert.Equal(t, headers, d.Headers())

	assert.Error(t, d.Ack(true))
	assert.Error(t, d.Nack(true, true))
	assert.Error(t, d.Reject(true))
}

func TestPkg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	timeNow := time.Now()
	dMock := NewMockDelivery(ctrl)

	pkg := &inAmqpPkg{
		delivery:   dMock,
		receivedAt: timeNow,
		origin:     "origin",
	}

	headers := amqp.Table{
		"uid": "xxx",
	}
	dMock.EXPECT().Headers().Return(headers).Times(3)
	dMock.EXPECT().Timestamp().Return(timeNow)

	assert.Equal(t, "xxx", pkg.UID())
	assert.Equal(t, "origin", pkg.Origin())
	assert.Equal(t, timeNow, pkg.ReceivedAt())
	assert.Equal(t, timeNow, pkg.PublishedAt())
	assert.Equal(t, map[string]interface{}{
		"uid": "xxx",
	}, pkg.Headers())
	assert.Nil(t, pkg.Attributes())

	dMock.EXPECT().Ack(true).Return(nil)
	assert.NoError(t, pkg.Ack(WithMultiple()))

	dMock.EXPECT().Nack(true, true).Return(nil)
	assert.NoError(t, pkg.Nack(WithMultiple(), WithRequeue()))

	dMock.EXPECT().Reject(true).Return(nil)
	assert.NoError(t, pkg.Reject(WithRequeue()))
}
