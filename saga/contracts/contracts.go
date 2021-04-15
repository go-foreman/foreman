package contracts

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga"
)
const (
	systemGroup scheme.Group = "systemSaga"
)
func init() {
	contractsList := []scheme.Object{
		&StartSagaCommand{},
		&RecoverSagaCommand{},
		&CompensateSagaCommand{},
		&SagaCompletedEvent{},
		&SagaChildCompletedEvent{},
	}
	scheme.KnownTypesRegistryInstance.AddKnownTypes(systemGroup, contractsList...)
}

type StartSagaCommand struct {
	message.ObjectMeta
	SagaId   string      `json:"saga_id" mapstructure:"saga_id"`
	ParentId string      `json:"parent_id" mapstructure:"parent_id"`
	Saga     saga.Saga   `json:"saga"`
}

type RecoverSagaCommand struct {
	message.ObjectMeta
	SagaId string `json:"saga_id"`
}

type CompensateSagaCommand struct {
	message.ObjectMeta
	SagaId string `json:"saga_id"`
}

type SagaCompletedEvent struct {
	message.ObjectMeta
	SagaId string `json:"saga_id"`
}

type SagaChildCompletedEvent struct {
	message.ObjectMeta
	SagaId string `json:"saga_id"`
}
