package contracts

import "github.com/go-foreman/foreman/pkg/runtime/scheme"

func init() {
	contractsList := []interface{}{
		&StartSagaCommand{},
		&RecoverSagaCommand{},
		&CompensateSagaCommand{},
		&SagaCompletedEvent{},
		&SagaChildCompletedEvent{},
	}
	scheme.KnownTypesRegistryInstance.RegisterTypes(contractsList...)
}

type StartSagaCommand struct {
	SagaId   string      `json:"saga_id" mapstructure:"saga_id"`
	ParentId string      `json:"parent_id" mapstructure:"parent_id"`
	SagaName string      `json:"saga_name" mapstructure:"saga_name"`
	Saga     interface{} `json:"saga"`
}

type RecoverSagaCommand struct {
	SagaId string `json:"saga_id"`
}

type CompensateSagaCommand struct {
	SagaId string `json:"saga_id"`
}

type SagaCompletedEvent struct {
	SagaId string `json:"saga_id"`
}

type SagaChildCompletedEvent struct {
	SagaId string `json:"saga_id"`
}
