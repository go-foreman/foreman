package amqp

import (
	"github.com/go-foreman/foreman/pubsub/transport"
	"github.com/pkg/errors"
)

type consumeOptions struct {
	Exclusive     bool
	NoLocal       bool
	NoWait        bool
	PrefetchCount uint
}

func convertConsumeOptsType(options interface{}) (*consumeOptions, error) {
	opts, ok := options.(*consumeOptions)

	if !ok {
		return nil, errors.Errorf("this option must be called on amqp.consumeOptions type")
	}

	return opts, nil
}

func convertSendOptsType(options interface{}) (*sendOptions, error) {
	opts, ok := options.(*sendOptions)

	if !ok {
		return nil, errors.Errorf("this option must be called on amqp.sendOptions type")
	}

	return opts, nil
}

func WithQosPrefetchCount(limit uint) transport.ConsumeOpt {
	return func(options interface{}) error {
		opts, err := convertConsumeOptsType(options)

		if err != nil {
			return errors.Wrap(err, "calling WithQosPrefetchCount opt")
		}
		opts.PrefetchCount = limit
		return nil
	}
}

func WithExclusive() transport.ConsumeOpt {
	return func(options interface{}) error {
		opts, err := convertConsumeOptsType(options)

		if err != nil {
			return errors.Wrap(err, "calling WithExclusive opt")
		}

		opts.Exclusive = true

		return nil
	}
}

func WithNoLocal() transport.ConsumeOpt {
	return func(options interface{}) error {
		opts, err := convertConsumeOptsType(options)

		if err != nil {
			return errors.Wrap(err, "calling WithNoLocal opt")
		}

		opts.NoLocal = true

		return nil
	}
}

func WithNoWait() transport.ConsumeOpt {
	return func(options interface{}) error {
		opts, err := convertConsumeOptsType(options)

		if err != nil {
			return errors.Wrap(err, "calling WithNoWait opt")
		}

		opts.NoWait = true

		return nil
	}
}

type sendOptions struct {
	Mandatory bool
	Immediate bool
}

func WithMandatory() transport.SendOpt {
	return func(options interface{}) error {
		opts, err := convertSendOptsType(options)

		if err != nil {
			return errors.Wrap(err, "calling WithMandatory opt")
		}

		opts.Mandatory = true

		return nil
	}
}

func WithImmediate() transport.SendOpt {
	return func(options interface{}) error {
		opts, err := convertSendOptsType(options)

		if err != nil {
			return errors.Wrap(err, "calling WithImmediate opt")
		}

		opts.Immediate = true

		return nil
	}
}
