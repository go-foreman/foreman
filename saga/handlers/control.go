package handlers

import (
	"context"

	log "github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/runtime/scheme"
	sagaPkg "github.com/go-foreman/foreman/saga"
	"github.com/go-foreman/foreman/saga/contracts"
	"github.com/go-foreman/foreman/saga/mutex"
	"github.com/pkg/errors"
)

func NewSagaControlHandler(sagaStore sagaPkg.Store, mutex mutex.Mutex, sagaRegistry scheme.KnownTypesRegistry, sagaUIDSvc sagaPkg.SagaUIDService) *SagaControlHandler {
	return &SagaControlHandler{typesRegistry: sagaRegistry, store: sagaStore, mutex: mutex, sagaUIDSvc: sagaUIDSvc}
}

type SagaControlHandler struct {
	typesRegistry scheme.KnownTypesRegistry
	store         sagaPkg.Store
	mutex         mutex.Mutex
	sagaUIDSvc    sagaPkg.SagaUIDService
}

func (h SagaControlHandler) Handle(execCtx execution.MessageExecutionCtx) error {
	var (
		sagaInstance sagaPkg.Instance
		sagaCtx      sagaPkg.SagaContext
		err          error
	)

	ctx := execCtx.Context()
	msg := execCtx.Message()
	logger := execCtx.Logger()

	switch cmd := msg.Payload().(type) {
	case *contracts.StartSagaCommand:
		logger.Logf(log.DebugLevel, "creating saga %s", cmd.SagaUID)

		sagaInstance, err = h.createSaga(cmd)
		if err != nil {
			return errors.WithStack(err)
		}

		lock, err := h.mutex.Lock(ctx, cmd.SagaUID)
		if err != nil {
			return errors.Wrap(err, "locking saga")
		}

		defer func() {
			if err := lock.Release(ctx); err != nil {
				logger.Log(log.ErrorLevel, err.Error())
			}
		}()

		if err := h.store.Create(ctx, sagaInstance); err != nil {
			return errors.Wrapf(err, "saving created saga `%s` with id %s to store", cmd.Saga.GroupKind().String(), cmd.SagaUID)
		}

		logger.Logf(log.DebugLevel, "saga %s created in store", cmd.SagaUID)

		sagaCtx = sagaPkg.NewSagaCtx(execCtx, sagaInstance)

		if err := sagaInstance.Start(sagaCtx); err != nil {
			return errors.Wrapf(err, "starting saga `%s`", sagaInstance.UID())
		}

	case *contracts.RecoverSagaCommand:
		lock, err := h.mutex.Lock(ctx, cmd.SagaUID)
		if err != nil {
			return errors.Wrapf(err, "locking saga %s", cmd.SagaUID)
		}

		defer func() {
			if err := lock.Release(ctx); err != nil {
				logger.Log(log.ErrorLevel, err.Error())
			}
		}()

		sagaInstance, err = h.fetchSaga(ctx, cmd.SagaUID)

		if err != nil {
			return errors.WithStack(err)
		}

		if !sagaInstance.Status().Failed() || sagaInstance.Status().Completed() || sagaInstance.Status().Recovering() || sagaInstance.Status().Compensating() {
			logger.Logf(log.InfoLevel, "Saga %s has status %s, you can't start recovering the process", sagaInstance.UID(), sagaInstance.Status())
			return nil
		}

		sagaCtx = sagaPkg.NewSagaCtx(execCtx, sagaInstance)

		if err := sagaInstance.Recover(sagaCtx); err != nil {
			return errors.Wrapf(err, "recovering saga `%s`", sagaInstance.UID())
		}

	case *contracts.CompensateSagaCommand:
		lock, err := h.mutex.Lock(ctx, cmd.SagaUID)
		if err != nil {
			return errors.Wrap(err, "locking saga")
		}

		defer func() {
			if err := lock.Release(ctx); err != nil {
				logger.Log(log.ErrorLevel, err.Error())
			}
		}()

		sagaInstance, err = h.fetchSaga(ctx, cmd.SagaUID)

		if err != nil {
			return errors.WithStack(err)
		}

		if !sagaInstance.Status().Failed() || sagaInstance.Status().Compensating() {
			logger.Logf(log.InfoLevel, "Saga `%s` has status `%s`, you can't compensate the process", sagaInstance.UID(), sagaInstance.Status())
			return nil
		}

		sagaCtx = sagaPkg.NewSagaCtx(execCtx, sagaInstance)

		if err := sagaInstance.Compensate(sagaCtx); err != nil {
			return errors.Wrapf(err, "compensating saga `%s`", sagaInstance.UID())
		}

	default:
		return errors.Errorf("unknown command type `%s` for SagaControlHandler. Supported: StartSagaCommand, RecoverSagaCommand, CompensateSagaCommand", msg.Payload().GroupKind().String())
	}

	sagaInstance.AddHistoryEvent(msg.Payload(), &sagaPkg.AddHistoryEvent{Origin: msg.Origin(), TraceUID: msg.UID()})

	for _, delivery := range sagaCtx.Deliveries() {
		h.sagaUIDSvc.AddSagaId(msg.Headers(), sagaCtx.SagaInstance().UID())
		outcomingMessage := message.NewOutcomingMessage(delivery.Payload, message.WithHeaders(msg.Headers()))

		if err := execCtx.Send(outcomingMessage, delivery.Options...); err != nil {
			logger.Logf(log.ErrorLevel, "sending delivery for saga %s. Delivery: (%v). %s", sagaCtx.SagaInstance().UID(), delivery, err)
			return errors.Wrapf(err, "sending delivery for saga %s. Delivery: (%v)", sagaCtx.SagaInstance().UID(), delivery)
		}
		sagaCtx.SagaInstance().AddHistoryEvent(delivery.Payload, nil)
	}

	return h.store.Update(ctx, sagaInstance)
}

//saga is map[string]interface{} on this step
func (h SagaControlHandler) createSaga(startCmd *contracts.StartSagaCommand) (sagaPkg.Instance, error) {
	if startCmd.SagaUID == "" {
		return nil, errors.Errorf("sagaId is empty")
	}

	if startCmd.Saga == nil {
		return nil, errors.Errorf("saga payload is nil")
	}

	saga, ok := startCmd.Saga.(sagaPkg.Saga)

	if !ok {
		return nil, errors.Errorf("error asserting that startCmd.Saga is Saga type")
	}

	return sagaPkg.NewSagaInstance(startCmd.SagaUID, startCmd.ParentUID, saga), nil
}

func (h SagaControlHandler) fetchSaga(ctx context.Context, sagaId string) (sagaPkg.Instance, error) {
	sagaInstance, err := h.store.GetById(ctx, sagaId)

	if err != nil {
		return nil, errors.Wrapf(err, "fetching saga instance `%s` from store", sagaId)
	}

	if sagaInstance == nil {
		return nil, errors.Errorf("Saga instance `%s` not found", sagaId)
	}

	return sagaInstance, nil
}
