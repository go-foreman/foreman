package contracts

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
)

const (
	SystemGroup scheme.Group = "systemSaga"
)

// StartSagaCommand once received will create SagaInstance, save it to Store and Start()
type StartSagaCommand struct {
	message.ObjectMeta
	SagaId   string      `json:"saga_id"`
	ParentId string      `json:"parent_id"`
	Saga     message.Object   `json:"saga"`
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
