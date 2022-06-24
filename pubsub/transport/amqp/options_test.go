package amqp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendOpts(t *testing.T) {
	type someOther struct {
	}

	opts := someOther{}

	withMandatory := WithMandatory()

	err := withMandatory(opts)
	assert.Error(t, err)
	assert.EqualError(t, err, "calling WithMandatory opt: this option must be called on amqp.sendOptions type")

	withImmediate := WithImmediate()

	err = withImmediate(opts)
	assert.Error(t, err)
	assert.EqualError(t, err, "calling WithImmediate opt: this option must be called on amqp.sendOptions type")
}

func TestConsumeOpts(t *testing.T) {
	type someOther struct {
	}

	opts := someOther{}

	o := WithExclusive()

	err := o(opts)
	assert.Error(t, err)
	assert.EqualError(t, err, "calling WithExclusive opt: this option must be called on amqp.consumeOptions type")

	o = WithExclusive()

	err = o(opts)
	assert.Error(t, err)
	assert.EqualError(t, err, "calling WithExclusive opt: this option must be called on amqp.consumeOptions type")

	o = WithNoLocal()

	err = o(opts)
	assert.Error(t, err)
	assert.EqualError(t, err, "calling WithNoLocal opt: this option must be called on amqp.consumeOptions type")

	o = WithNoWait()

	err = o(opts)
	assert.Error(t, err)
	assert.EqualError(t, err, "calling WithNoWait opt: this option must be called on amqp.consumeOptions type")

	o = WithQosPrefetchCount(100)

	err = o(opts)
	assert.Error(t, err)
	assert.EqualError(t, err, "calling WithQosPrefetchCount opt: this option must be called on amqp.consumeOptions type")
}
