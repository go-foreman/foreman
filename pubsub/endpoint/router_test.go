package endpoint

import (
	"context"
	"testing"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/stretchr/testify/assert"
)

func TestRouter(t *testing.T) {
	testEndpoint := &testEndpoint{}

	router := NewRouter()
	router.RegisterEndpoint(testEndpoint, &testObj{})
	endpoints := router.Route(&testObj{})

	assert.Len(t, endpoints, 1)
	assert.Same(t, testEndpoint, endpoints[0])

	endpoints = router.Route(&anotherObj{})
	assert.Empty(t, endpoints)

}

type testObj struct {
	message.ObjectMeta
	Data string
}

type anotherObj struct {
	message.ObjectMeta
}

type testEndpoint struct {
}

func (t testEndpoint) Name() string {
	//TODO implement me
	panic("implement me")
}

func (t testEndpoint) Send(ctx context.Context, message *message.OutcomingMessage, options ...DeliveryOption) error {
	//TODO implement me
	panic("implement me")
}
