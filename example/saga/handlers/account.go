package handlers

import (
	"fmt"
	"github.com/kopaygorodsky/brigadier/example/saga/usecase/account/contracts"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message/execution"
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type Account struct {
	uid                string
	email              string
	confirmationSentAt time.Time
}

type AccountHandler struct {
	lock sync.Mutex
	runtimeDb map[string]*Account
}

func NewAccountHandler() *AccountHandler {
	return &AccountHandler{runtimeDb: make(map[string]*Account)}
}

func (h *AccountHandler) RegisterAccount(execCtx execution.MessageExecutionCtx) error {
	receivedMsg := execCtx.Message()
	registerAccountCmd, _ := receivedMsg.Payload.(*contracts.RegisterAccountCmd)

	time.Sleep(time.Second * 3)
	if rand.Intn(10)%2 != 0 {
		failedEvent := &contracts.RegistrationFailed{UID: registerAccountCmd.UID, Reason: "idk, some error happened :)"}
		return execCtx.Send(message.NewEventMessage(failedEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(failedEvent.Reason)))
	}

	account := &Account{
		uid:   registerAccountCmd.UID,
		email: registerAccountCmd.Email,
	}

	h.saveAccount(account)

	time.Sleep(time.Second * 3)
	successEvent := &contracts.AccountedRegistered{UID: registerAccountCmd.UID}
	return execCtx.Send(message.NewEventMessage(successEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(fmt.Sprintf("Account %s was registered", registerAccountCmd.UID))))
}

func (h *AccountHandler) SendConfirmation(execCtx execution.MessageExecutionCtx) error {
	receivedMsg := execCtx.Message()
	sendConfirmationCmd, _ := receivedMsg.Payload.(*contracts.SendConfirmationCmd)

	account, exists := h.getAccount(sendConfirmationCmd.UID)
	if !exists {
		failedEvent := &contracts.RegistrationFailed{UID: sendConfirmationCmd.UID, Reason: "account does not exist"}
		return execCtx.Send(message.NewEventMessage(failedEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(failedEvent.Reason)))
	}

	//trying to simulate user who confirms the account. This action should be done in some API handler... A bit lazy to simulate API server now.
	go func(execCtx execution.MessageExecutionCtx, uid string, headers message.Headers) {
		time.Sleep(time.Second * 30)
		accountConfirmedEvent := &contracts.AccountedRegistered{UID: sendConfirmationCmd.UID}
		//we can reuse context to send this message to the endpoint
		execCtx.Send(message.NewEventMessage(accountConfirmedEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(fmt.Sprintf("Confirmation to %s sent", sendConfirmationCmd.Email))))
	}(execCtx, sendConfirmationCmd.UID, receivedMsg.Headers)

	time.Sleep(time.Second * 4)
	account.confirmationSentAt = time.Now()
	successEvent := &contracts.ConfirmationSent{UID: sendConfirmationCmd.UID}
	return execCtx.Send(message.NewEventMessage(successEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(fmt.Sprintf("Confirmation to %s sent", sendConfirmationCmd.Email))))
}

func (h *AccountHandler) getAccount(uid string) (*Account, bool) {
	acc, exists := h.runtimeDb[uid]
	return acc, exists
}

func (h *AccountHandler) saveAccount(acc *Account) {
	h.lock.Lock()
	h.runtimeDb[acc.uid] = acc
	h.lock.Unlock()
}
