package handlers

import (
	log "github.com/kopaygorodsky/brigadier/pkg/log"
	busErrs "github.com/kopaygorodsky/brigadier/pkg/pubsub/errors"
	sagaPkg "github.com/kopaygorodsky/brigadier/pkg/saga"
	sagaMutex "github.com/kopaygorodsky/brigadier/pkg/saga/mutex"

	"fmt"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message"
	"github.com/kopaygorodsky/brigadier/pkg/pubsub/message/execution"
	"github.com/kopaygorodsky/brigadier/pkg/runtime/scheme"
	"github.com/kopaygorodsky/brigadier/pkg/saga/contracts"
	"github.com/pkg/errors"
	"time"
)

type SagaEventsHandler struct {
	sagaStore     sagaPkg.Store
	idExtractor   sagaPkg.IdExtractor
	typesRegistry scheme.KnownTypesRegistry
	mutex         sagaMutex.Mutex
	logger        log.Logger
}

func NewEventsHandler(sagaStore sagaPkg.Store, mutex sagaMutex.Mutex, sagaRegistry scheme.KnownTypesRegistry, extractor sagaPkg.IdExtractor, logger log.Logger) *SagaEventsHandler {
	return &SagaEventsHandler{sagaStore: sagaStore, idExtractor: extractor, typesRegistry: sagaRegistry, mutex: mutex, logger: logger}
}

func (e SagaEventsHandler) Handle(execCtx execution.MessageExecutionCtx) error {
	msg := execCtx.Message()
	ctx := execCtx.Context()

	if msg.Type != "event" {
		return errors.Errorf("Received message %s is not an event type", msg.ID)
	}

	sagaId, err := e.idExtractor.ExtractSagaId(msg)

	if err != nil {
		return errors.Wrapf(err, "Error extracting saga id from message %s", msg.ID)
	}

	//lock saga so nobody can process events for this saga in another consumer's replicas
	if err := e.mutex.Lock(ctx, sagaId); err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		if err := e.mutex.Release(ctx, sagaId); err != nil {
			e.logger.Log(log.ErrorLevel, err)
		}
	}()

	sagaInstance, err := e.sagaStore.GetById(ctx, sagaId)

	if err != nil {
		return errors.Wrapf(err, "Error retrieving saga %s from store", sagaId)
	}

	if sagaInstance == nil {
		return errors.Errorf("Saga %s not found", sagaId)
	}

	if sagaInstance.Completed() {
		return busErrs.WithStatusErr(busErrs.NoRetry, errors.Errorf("Saga %s already completed", sagaId))
	}

	saga := sagaInstance.Saga()
	saga.Init()

	sagaCtx := sagaPkg.NewSagaCtx(execCtx, sagaInstance)

	if handler, exists := saga.EventHandlers()[msg.Name]; exists {

		if err := handler(sagaCtx); err != nil {
			execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("error handling saga event %s from message %s: %s", msg.Name, msg.ID, err))
			return errors.Wrapf(err, "error handling event %s from message %s", msg.Name, msg.ID)
		}

		for _, delivery := range sagaCtx.Deliveries() {
			if err := execCtx.Send(delivery.Message, delivery.Options...); err != nil {
				execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("error sending delivery for saga %s. Delivery: (%v). %s", sagaCtx.SagaInstance().ID(), delivery, err))
				return errors.Wrapf(err, "error sending delivery for saga %s. Delivery: (%v)", sagaCtx.SagaInstance().ID(), delivery)
			}
			sagaCtx.SagaInstance().AttachEvent(sagaPkg.HistoryEvent{Metadata: delivery.Message.Metadata, Payload: delivery.Message.Payload, CreatedAt: time.Now()})
		}
	} else {
		e.logger.Logf(log.WarnLevel, "No handler defined for event %s from message %s", msg.Name, msg.ID)
	}

	sagaInstance.AttachEvent(sagaPkg.HistoryEvent{Metadata: msg.Metadata, Payload: msg.Payload, CreatedAt: time.Now(), OriginSource: msg.OriginSource, SagaStatus: sagaInstance.Status(), Description: msg.Description})

	if err := e.sagaStore.Update(ctx, sagaInstance); err != nil {
		return errors.Wrapf(err, "error saving saga's %s state to db", sagaInstance.ID())
	}

	//sending an event about saga completion to parent if it exists and to all regular handlers.
	if sagaInstance.Completed() {
		var msg = &message.Message{}
		//if parent exists - we should forward this event to parent saga
		if sagaInstance.ParentID() != "" {
			msg = message.NewEventMessage(contracts.SagaChildCompletedEvent{SagaId: sagaInstance.ID()})
			msg.Headers[sagaPkg.SagaIdKey] = sagaInstance.ParentID()
			return execCtx.Send(msg)
		}
	}

	return nil
}
