package pkg

func WithRequeue() AcknowledgmentOption {
	return func(options map[string]interface{}) {
		options["requeue"] = true
	}
}

func WithMultiple() AcknowledgmentOption {
	return func(options map[string]interface{}) {
		options["multiple"] = true
	}
}
