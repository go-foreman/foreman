package handlers

import (
	"fmt"
	"github.com/go-foreman/foreman/example/saga/usecase/account/contracts"
	"github.com/go-foreman/foreman/pkg/log"
	"github.com/go-foreman/foreman/pkg/pubsub/message"
	"github.com/go-foreman/foreman/pkg/pubsub/message/execution"
	"io/ioutil"
	"math/rand"
	"path"
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
	sync.Mutex
	runtimeDb        map[string]*Account
	logger           log.Logger
	confirmationsDir string
}

func NewAccountHandler(logger log.Logger, confirmationsDir string) (*AccountHandler, error) {
	return &AccountHandler{runtimeDb: make(map[string]*Account), logger: logger, confirmationsDir: confirmationsDir}, nil
}

func (h *AccountHandler) RegisterAccount(execCtx execution.MessageExecutionCtx) error {
	receivedMsg := execCtx.Message()
	registerAccountCmd, _ := receivedMsg.Payload.(*contracts.RegisterAccountCmd)

	time.Sleep(time.Second * 1)
	if rand.Intn(10)%2 != 0 {
		failedEvent := &contracts.RegistrationFailed{UID: registerAccountCmd.UID, Reason: "idk, some error happened :)"}
		return execCtx.Send(message.NewEventMessage(failedEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(failedEvent.Reason)))
	}

	account, exists := h.getAccount(registerAccountCmd.UID)
	if exists {
		failedEvent := &contracts.RegistrationFailed{UID: registerAccountCmd.UID, Reason: "account already created"}
		return execCtx.Send(message.NewEventMessage(failedEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(failedEvent.Reason)))
	}

	account = &Account{
		uid:   registerAccountCmd.UID,
		email: registerAccountCmd.Email,
	}

	h.saveAccount(account)
	successEvent := &contracts.AccountRegistered{UID: registerAccountCmd.UID}
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

	account.confirmationSentAt = time.Now()

	//only for this use-case we need to write sagaId into file, usually it will be stored in db->
	//user confirms registration with a token, we should look for an account in db by token, take sagaId and finishing our process.
	v, ok := receivedMsg.Headers["sagaId"]
	if !ok {
		panic("sagaId header is empty")
	}

	sagaId, ok := v.(string)
	if !ok {
		panic("sagaId is not a string")
	}

	err := ioutil.WriteFile(path.Join(h.confirmationsDir, sagaId), []byte(sendConfirmationCmd.UID), 0777)
	if err != nil {
		failedEvent := &contracts.RegistrationFailed{UID: sendConfirmationCmd.UID, Reason: err.Error()}
		return execCtx.Send(message.NewEventMessage(failedEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(failedEvent.Reason)))
	}

	return nil

	//don't need to send an event about confirmation being sent. Need to handle only sending failure if such exists.
	//successEvent := &contracts.ConfirmationSent{UID: sendConfirmationCmd.UID}
	//return execCtx.Send(message.NewEventMessage(successEvent, message.WithHeaders(receivedMsg.Headers), message.WithDescription(fmt.Sprintf("Confirmation to %s sent", sendConfirmationCmd.Email))))
}

func (h *AccountHandler) getAccount(uid string) (*Account, bool) {
	h.Lock()
	defer h.Unlock()
	acc, exists := h.runtimeDb[uid]
	return acc, exists
}

func (h *AccountHandler) saveAccount(acc *Account) {
	h.Lock()
	defer h.Unlock()
	h.runtimeDb[acc.uid] = acc
}
