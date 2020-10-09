package contracts

import (
	"github.com/kopaygorodsky/brigadier/example/saga/usecase"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
)

func init() {
	contractsList := []interface{}{
		&RegisterAccountCmd{},
		&AccountedRegistered{},
		&RegistrationFailed{},
		&SendConfirmationCmd{},
		&ConfirmationSent{},
		&ConfirmationSendingFailed{},
		&AccountConfirmed{},
	}

	scheme.KnownTypesRegistryInstance.RegisterTypes(contractsList...)
	usecase.DefaultSagasCollection.RegisterContracts(contractsList...)
}

type RegisterAccountCmd struct {
	UID   string `json:"uid"`
	Email string `json:"name"`
}

type AccountedRegistered struct {
	UID string `json:"uid"`
}

type RegistrationFailed struct {
	UID    string `json:"uid"`
	Reason string `json:"reason"`
}

type SendConfirmationCmd struct {
	UID   string `json:"uid"`
	Email string `json:"email"`
}

type ConfirmationSent struct {
	UID string `json:"uid"`
}

type ConfirmationSendingFailed struct {
	UID    string `json:"uid"`
	Reason string `json:"reason"`
}

type AccountConfirmed struct {
	UID string `json:"uid"`
}
