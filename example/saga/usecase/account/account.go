package account

import (
	"fmt"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase/account/contracts"
	"github.com/kopaygorodsky/brigadier/pkg/log"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/saga"
)

func init() {
	usecase.DefaultSagasCollection.AddSaga(&RegisterAccountSaga{})
}

type RegisterAccountSaga struct {
	saga.BaseSaga
	UID          string
	Email        string
	Password     string
	RetriesLimit int
}

func (r *RegisterAccountSaga) Init() {
	r.
		AddEventHandler(&contracts.AccountRegistered{}, r.AccountRegistered).
		AddEventHandler(&contracts.RegistrationFailed{}, r.RegistrationFailed).
		AddEventHandler(&contracts.ConfirmationSent{}, r.ConfirmationSent).
		AddEventHandler(&contracts.ConfirmationSendingFailed{}, r.ConfirmationSendingFailed).
		AddEventHandler(&contracts.AccountConfirmed{}, r.AccountConfirmed)
}

func (r RegisterAccountSaga) Start(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Starting saga %s", execCtx.SagaInstance().ID()))
	execCtx.Dispatch(message.NewCommandMessage(contracts.RegisterAccountCmd{
		UID:   r.UID,
		Email: r.Email,
	}))
	return nil
}

func (r RegisterAccountSaga) Compensate(execCtx saga.SagaContext) error {
	return nil
}

func (r RegisterAccountSaga) Recover(execCtx saga.SagaContext) error {
	return nil
}

func (r *RegisterAccountSaga) AccountRegistered(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Account registration successful. Sending confirmation to %s", r.Email))
	execCtx.Dispatch(message.NewCommandMessage(contracts.SendConfirmationCmd{
		UID:   r.UID,
		Email: r.Email,
	}))
	return nil
}

func (r *RegisterAccountSaga) RegistrationFailed(execCtx saga.SagaContext) error {
	msg, _ := execCtx.Message().Payload.(*contracts.RegistrationFailed)

	//here you can do retries if you want
	execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("Account registration failed. Reason %s", msg.Reason))

	if r.RetriesLimit > 0 {
		execCtx.Dispatch(message.NewCommandMessage(contracts.RegisterAccountCmd{
			UID:   r.UID,
			Email: r.Email,
		}))
		r.RetriesLimit--
		return nil
	}

	execCtx.SagaInstance().Fail()
	return nil
}

func (r *RegisterAccountSaga) ConfirmationSent(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Account confirmation sent out to %s", r.Email))
	return nil
}

func (r *RegisterAccountSaga) ConfirmationSendingFailed(execCtx saga.SagaContext) error {
	msg, _ := execCtx.Message().Payload.(*contracts.ConfirmationSendingFailed)

	//some retry logic
	execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("Failed sending account confirmation for %s. Reason %s", r.Email, msg.Reason))
	execCtx.SagaInstance().Fail()

	return nil
}

func (r *RegisterAccountSaga) AccountConfirmed(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Account %s confirmed by %s", r.UID, r.Email))
	execCtx.SagaInstance().Completed()
	return nil
}
