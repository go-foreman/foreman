package handlers

import (
	"context"
	"time"

	log "github.com/go-foreman/foreman/log"
	sagaPkg "github.com/go-foreman/foreman/saga"
	sagaMutex "github.com/go-foreman/foreman/saga/mutex"

	"fmt"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga/contracts"
	"github.com/pkg/errors"
)

type SagaEventsHandler struct {
	sagaStore  sagaPkg.Store
	sagaUIDSvc sagaPkg.SagaUIDService
	scheme     scheme.KnownTypesRegistry
	mutex      sagaMutex.Mutex
}

func NewEventsHandler(sagaStore sagaPkg.Store, mutex sagaMutex.Mutex, scheme scheme.KnownTypesRegistry, extractor sagaPkg.SagaUIDService) *SagaEventsHandler {
	return &SagaEventsHandler{sagaStore: sagaStore, sagaUIDSvc: extractor, scheme: scheme, mutex: mutex}
}

func (e SagaEventsHandler) Handle(execCtx execution.MessageExecutionCtx) error {
	msg := execCtx.Message()
	ctx := execCtx.Context()
	logger := execCtx.Logger()
	msgGK := msg.Payload().GroupKind().String()

	sagaId, err := e.sagaUIDSvc.ExtractSagaUID(msg.Headers())

	if err != nil {
		return errors.Wrapf(err, "extracting saga id from message %s", msg.UID())
	}

	//lock saga so nobody can process events for this saga in another consumer's replicas
	lock, err := e.mutex.Lock(ctx, sagaId)
	if err != nil {
		return errors.WithStack(err)
	}

	logger.Logf(log.DebugLevel, "locked saga %s", sagaId)

	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		if err := lock.Release(releaseCtx); err != nil {
			logger.Log(log.ErrorLevel, err.Error())
		}
		logger.Logf(log.DebugLevel, "released saga %s", sagaId)
	}()

	sagaInstance, err := e.sagaStore.GetById(ctx, sagaId)
	logger.Logf(log.DebugLevel, "loaded saga %s", sagaId)

	if err != nil {
		return errors.Wrapf(err, "Error retrieving saga %s from store", sagaId)
	}

	if sagaInstance == nil {
		return errors.Errorf("Saga %s not found", sagaId)
	}

	if sagaInstance.Status().Completed() {
		return errors.Errorf("Saga %s already completed", sagaId)
	}

	saga := sagaInstance.Saga()
	saga.SetSchema(e.scheme)
	saga.Init()

	sagaCtx := sagaPkg.NewSagaCtx(execCtx, sagaInstance)

	if handler, exists := saga.EventHandlers()[msg.Payload().GroupKind()]; exists {

		if err := handler(sagaCtx); err != nil {
			logger.Log(log.ErrorLevel, fmt.Sprintf("error handling saga event %s from message %s: %s", msgGK, msg.UID(), err))
			return errors.Wrapf(err, "handling event %s from message %s", msgGK, msg.UID())
		}

		for _, delivery := range sagaCtx.Deliveries() {
			e.sagaUIDSvc.AddSagaId(execCtx.Message().Headers(), sagaInstance.UID())
			outcomingMsg := message.NewOutcomingMessage(delivery.Payload, message.WithHeaders(execCtx.Message().Headers()))

			if err := execCtx.Send(outcomingMsg, delivery.Options...); err != nil {
				logger.Log(log.ErrorLevel, fmt.Sprintf("error sending delivery for saga %s. Delivery: (%v). %s", sagaCtx.SagaInstance().UID(), delivery, err))
				return errors.Wrapf(err, "sending delivery for saga %s. Delivery: (%v)", sagaCtx.SagaInstance().UID(), delivery)
			}
		}
	} else {
		logger.Logf(log.WarnLevel, "no handler defined for event %s from message %s", msgGK, msg.UID())
	}

	//write received event into history
	sagaInstance.AddHistoryEvent(msg.Payload(), &sagaPkg.AddHistoryEvent{
		TraceUID: msg.UID(),
		Origin:   msg.Origin(),
	})

	//just to remember what we sent out
	for _, ev := range sagaCtx.Deliveries() {
		sagaInstance.AddHistoryEvent(ev.Payload, nil)
	}

	if err := e.sagaStore.Update(ctx, sagaInstance); err != nil {
		return errors.Wrapf(err, "error saving saga's %s state to db", sagaInstance.UID())
	}

	//sending an event about saga completion to parent if it exists and to all regular handlers.
	if sagaInstance.Status().Completed() {
		//if parent exists - we should forward this event to parent saga
		if sagaInstance.ParentID() != "" {
			e.sagaUIDSvc.AddSagaId(execCtx.Message().Headers(), sagaInstance.ParentID())

			return execCtx.Send(message.NewOutcomingMessage(&contracts.SagaChildCompletedEvent{SagaUID: sagaInstance.UID()}, message.WithHeaders(execCtx.Message().Headers())))
		}
	}

	return nil
}
