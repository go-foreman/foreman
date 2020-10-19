package contracts

import (
	"github.com/go-foreman/foreman/example/saga/usecase"
	"github.com/go-foreman/foreman/pkg/runtime/scheme"
)

func init() {
	contractsList := []interface{}{
		&RegisterAccountCmd{},
		&AccountRegistered{},
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

type AccountRegistered struct {
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
