package foreman

import (
	"testing"

	"github.com/go-foreman/foreman/testing/mocks/pubsub/dispatcher"
	"github.com/go-foreman/foreman/testing/mocks/pubsub/endpoint"
	"github.com/go-foreman/foreman/testing/mocks/pubsub/message/execution"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMessageBusConfigOptions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dispatcherMock := dispatcher.NewMockDispatcher(ctrl)
	msgExecFactoryMock := execution.NewMockMessageExecutionCtxFactory(ctrl)
	routerMock := endpoint.NewMockRouter(ctrl)
	componentMock := &aComponent{}

	c := &container{}

	opts := []ConfigOption{
		WithDispatcher(dispatcherMock),
		WithMessageExecutionFactory(msgExecFactoryMock),
		WithRouter(routerMock),
		WithComponents(componentMock),
	}

	for _, o := range opts {
		o(c)
	}

	assert.Same(t, c.messagesDispatcher, dispatcherMock)
	assert.Same(t, c.router, routerMock)
	assert.Equal(t, []Component{componentMock}, c.components)
	assert.Same(t, c.messageExuctionCtxFactory, msgExecFactoryMock)
}

type aComponent struct {
}

func (a aComponent) Init(b *MessageBus) error {
	//TODO implement me
	panic("implement me")
}
