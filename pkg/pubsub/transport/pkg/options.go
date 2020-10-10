package pkg

func WithRequeue() AcknowledgmentOption {
	return func(options *ackOpts) {
		options.requeue = true
	}
}

func WithMultiple() AcknowledgmentOption {
	return func(options *ackOpts) {
		options.multiple = true
	}
}
