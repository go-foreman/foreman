package dispatcher

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/stretchr/testify/assert"
	"testing"
)

type registerAccountCmd struct {
}

type sendConfirmationCmd struct {
}

type accountRegisteredEvent struct {
}

type confirmationSent struct {
}

func allExecutor(execCtx execution.MessageExecutionCtx) error {
	return nil
}

type service struct {
}

func (h *service) handle(execCtx execution.MessageExecutionCtx) error {
	return nil
}

var handler = &service{}

func TestRegisterCmdHandler(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.RegisterCmdHandler(&registerAccountCmd{}, handler.handle)
	dispatcher.RegisterCmdHandlerWithKey(scheme.WithStruct(sendConfirmationCmd{}), handler.handle)

	assert.Len(t, dispatcher.Match(message.NewCommandMessage(&registerAccountCmd{})), 1)
	assert.Len(t, dispatcher.Match(message.NewCommandMessage(&sendConfirmationCmd{})), 1)

	//check if it works with payload passed as non pointer struct
	assert.Len(t, dispatcher.Match(message.NewCommandMessage(sendConfirmationCmd{})), 1)

	//match some unknown struct
	assert.Len(t, dispatcher.Match(message.NewCommandMessage(&accountRegisteredEvent{})), 0)

	//let's try to add executor for all commands, it will be ignored.
	dispatcher.RegisterCmdHandlerWithKey(scheme.WithKey(MatchesAllKeys), allExecutor)

	//MatchesAllKeys is ignored for commands
	assert.Len(t, dispatcher.Match(message.NewCommandMessage(&registerAccountCmd{})), 1)

	//When you register handler as a method of some service, dispatcher allows to register only methods from a single instance.
	//Registration of same method of the same service will be ignored.
	newHandler := &service{}
	dispatcher.RegisterCmdHandler(&registerAccountCmd{}, newHandler.handle)
	assert.Len(t, dispatcher.Match(message.NewCommandMessage(&registerAccountCmd{})), 1)
}

func TestRegisterEventHandler(t *testing.T) {
	dispatcher := NewDispatcher()
	dispatcher.RegisterEventListener(&accountRegisteredEvent{}, handler.handle)
	dispatcher.RegisterEventListenerWithKey(scheme.WithStruct(&confirmationSent{}), handler.handle)

	assert.Len(t, dispatcher.Match(message.NewEventMessage(&accountRegisteredEvent{})), 1)
	assert.Len(t, dispatcher.Match(message.NewEventMessage(&confirmationSent{})), 1)

	//check if it works with payload passed as non pointer struct
	assert.Len(t, dispatcher.Match(message.NewEventMessage(confirmationSent{})), 1)

	//match some unknown struct
	assert.Len(t, dispatcher.Match(message.NewEventMessage(&sendConfirmationCmd{})), 0)

	//let's try to add executor for all commands, it will be ignored.
	dispatcher.RegisterEventListenerWithKey(scheme.WithKey(MatchesAllKeys), allExecutor)

	//MatchesAllKeys is registered successfully
	assert.Len(t, dispatcher.Match(message.NewEventMessage(&confirmationSent{})), 2)

	//When you register handler as a method of some service, dispatcher allows to register only methods from a single instance.
	//Registration of same method of the same service will be ignored.
	newHandler := &service{}
	dispatcher.RegisterEventListener(&confirmationSent{}, newHandler.handle) //you think it will register 3rd handlers, but it's not going to
	assert.Len(t, dispatcher.Match(message.NewEventMessage(&confirmationSent{})), 2)
}
