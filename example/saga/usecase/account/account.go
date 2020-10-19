package account

import (
	"fmt"
	"github.com/go-foreman/foreman/example/saga/usecase"
	"github.com/go-foreman/foreman/example/saga/usecase/account/contracts"
	"github.com/go-foreman/foreman/pkg/log"
	"github.com/go-foreman/foreman/pkg/pubsub/message"
	"github.com/go-foreman/foreman/pkg/saga"
)

func init() {
	usecase.DefaultSagasCollection.AddSaga(&RegisterAccountSaga{})
}

type RegisterAccountSaga struct {
	saga.BaseSaga
	UID                 string
	Email               string
	Password            string
	RetriesLimit        int
	confirmationWasSent bool
}

func (r *RegisterAccountSaga) Init() {
	r.
		AddEventHandler(&contracts.AccountRegistered{}, r.AccountRegistered).
		AddEventHandler(&contracts.RegistrationFailed{}, r.RegistrationFailed).
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

	execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("Account registration failed. Reason %s", msg.Reason))

	if r.RetriesLimit > 0 {
		r.RetriesLimit--
		execCtx.Dispatch(message.NewCommandMessage(contracts.RegisterAccountCmd{
			UID:   r.UID,
			Email: r.Email,
		}))
		return nil
	}

	execCtx.SagaInstance().Fail()
	return nil
}

func (r *RegisterAccountSaga) ConfirmationSendingFailed(execCtx saga.SagaContext) error {
	msg, _ := execCtx.Message().Payload.(*contracts.ConfirmationSendingFailed)

	//some retry logic if you want
	execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("Failed sending account confirmation for %s. Reason %s", r.Email, msg.Reason))
	execCtx.SagaInstance().Fail()

	return nil
}

func (r *RegisterAccountSaga) AccountConfirmed(execCtx saga.SagaContext) error {
	execCtx.LogMessage(log.InfoLevel, fmt.Sprintf("Account %s confirmed by %s", r.UID, r.Email))
	execCtx.SagaInstance().Complete()
	return nil
}
