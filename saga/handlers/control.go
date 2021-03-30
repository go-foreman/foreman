package handlers

import (
	log "github.com/go-foreman/foreman/log"
	sagaPkg "github.com/go-foreman/foreman/saga"

	"context"
	"fmt"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/pubsub/message/execution"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga/contracts"
	"github.com/go-foreman/foreman/saga/mutex"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"time"
)

func NewSagaControlHandler(sagaStore sagaPkg.Store, mutex mutex.Mutex, sagaRegistry scheme.KnownTypesRegistry, logger log.Logger) *SagaControlHandler {
	return &SagaControlHandler{typesRegistry: sagaRegistry, store: sagaStore, mutex: mutex, logger: logger}
}

type SagaControlHandler struct {
	typesRegistry scheme.KnownTypesRegistry
	store         sagaPkg.Store
	mutex         mutex.Mutex
	logger        log.Logger
}

func (h SagaControlHandler) Handle(execCtx execution.MessageExecutionCtx) error {
	var (
		sagaInstance sagaPkg.Instance
		sagaCtx      sagaPkg.SagaContext
		err          error
	)

	ctx := execCtx.Context()
	msg := execCtx.Message()

	switch cmd := msg.Payload.(type) {
	case *contracts.StartSagaCommand:
		sagaInstance, err = h.createSaga(cmd.SagaId, cmd.ParentId, cmd.SagaName, cmd.Saga)
		if err != nil {
			return errors.WithStack(err)
		}

		if err := h.store.Create(ctx, sagaInstance); err != nil {
			return errors.Wrapf(err, "error  saving created saga `%s` with id %s to store", cmd.SagaName, cmd.SagaId)
		}

		sagaCtx = sagaPkg.NewSagaCtx(execCtx, sagaInstance)

		if err := sagaInstance.Start(sagaCtx); err != nil {
			return errors.Wrapf(err, "error starting saga `%s`", sagaInstance.ID())
		}

	case *contracts.RecoverSagaCommand:
		if err := h.mutex.Lock(ctx, cmd.SagaId); err != nil {
			return errors.WithStack(err)
		}

		defer func() {
			if err := h.mutex.Release(ctx, cmd.SagaId); err != nil {
				h.logger.Log(log.ErrorLevel, err)
			}
		}()

		sagaInstance, err = h.fetchSaga(ctx, cmd.SagaId)

		if err != nil {
			return errors.WithStack(err)
		}

		if !sagaInstance.Status().Failed() || sagaInstance.Status().Completed() || sagaInstance.Status().Recovering() || sagaInstance.Status().Compensating() {
			h.logger.Logf(log.InfoLevel, "Saga `%s` has status %s, you can't start recovering the process", sagaInstance.Status(), sagaInstance.ID())
			return nil
		}

		sagaCtx = sagaPkg.NewSagaCtx(execCtx, sagaInstance)

		if err := sagaInstance.Recover(sagaCtx); err != nil {
			return errors.Wrapf(err, "error recovering saga `%s`", sagaInstance.ID())
		}

	case *contracts.CompensateSagaCommand:
		if err := h.mutex.Lock(ctx, cmd.SagaId); err != nil {
			return errors.WithStack(err)
		}

		defer func() {
			if err := h.mutex.Release(ctx, cmd.SagaId); err != nil {
				h.logger.Log(log.ErrorLevel, err)
			}
		}()

		sagaInstance, err = h.fetchSaga(ctx, cmd.SagaId)

		if err != nil {
			return errors.WithStack(err)
		}

		if !sagaInstance.Status().Failed() || sagaInstance.Status().Compensating() {
			h.logger.Logf(log.InfoLevel, "Saga `%s` has status `%s`, you can't compensate the process", sagaInstance.ID(), sagaInstance.Status())
			return nil
		}

		sagaCtx = sagaPkg.NewSagaCtx(execCtx, sagaInstance)

		if err := sagaInstance.Compensate(sagaCtx); err != nil {
			return errors.Wrapf(err, "error compensating saga `%s`", sagaInstance.ID())
		}

	default:
		return errors.Errorf("Unknown command type `%s` for SagaControlHandler. Supported: StartSagaCommand, RecoverSagaCommand, CompensateSagaCommand", msg.Type)
	}

	for _, delivery := range sagaCtx.Deliveries() {
		if err := execCtx.Send(delivery.Message, delivery.Options...); err != nil {
			execCtx.LogMessage(log.ErrorLevel, fmt.Sprintf("error sending delivery for saga %s. Delivery: (%v). %s", sagaCtx.SagaInstance().ID(), delivery, err))
			return errors.Wrapf(err, "error sending delivery for saga %s. Delivery: (%v)", sagaCtx.SagaInstance().ID(), delivery)
		}
		sagaCtx.SagaInstance().AttachEvent(sagaPkg.HistoryEvent{Metadata: delivery.Message.Metadata, Payload: delivery.Message.Payload, CreatedAt: time.Now()})
	}

	return h.store.Update(ctx, sagaInstance)
}

//saga is map[string]interface{} on this step
func (h SagaControlHandler) createSaga(sagaId, parentId, sagaName string, sagaDefinition interface{}) (sagaPkg.Instance, error) {

	if sagaId == "" {
		return nil, errors.Errorf("SagaId is empty")
	}

	if sagaName == "" {
		return nil, errors.Errorf("SagaName is empty")
	}

	if sagaDefinition == nil {
		return nil, errors.Errorf("Saga payload is nil")
	}

	sagaToCreate, err := h.typesRegistry.LoadType(scheme.WithKey(sagaName))

	if err != nil {
		return nil, errors.WithStack(err)
	}

	sagaType := h.typesRegistry.GetType(scheme.WithKey(sagaName))

	decoderConf := mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &sagaToCreate,
	}

	decoder, err := mapstructure.NewDecoder(&decoderConf)

	if err != nil {
		return nil, errors.Wrap(err, "error creating decoder")
	}

	if err := decoder.Decode(sagaDefinition); err != nil {
		return nil, message.WithDecoderErr(errors.Wrapf(err, "error decoding payload into saga  %s", sagaType.String()))
	}

	//kek, err := json.Marshal(saga)
	//
	//if err != nil {
	//	return nil, errors.WithStack(err)
	//}
	//
	//if err := json.Unmarshal(kek, &sagaToCreate); err != nil {
	//	return nil, errors.Wrapf(err, "Error decoding data from message payload interface{} to an original type %s", sagaType.Kind().String())
	//}

	sagaInterface, ok := sagaToCreate.(sagaPkg.Saga)

	if !ok {
		return nil, errors.Errorf("Error converting interface{} to Saga interface")
	}

	return sagaPkg.NewSagaInstance(sagaId, parentId, sagaInterface), nil
}

func (h SagaControlHandler) fetchSaga(ctx context.Context, sagaId string) (sagaPkg.Instance, error) {
	sagaInstance, err := h.store.GetById(ctx, sagaId)

	if err != nil {
		return nil, errors.Wrapf(err, "error fetching saga instance `%s` from store", sagaId)
	}

	if sagaInstance == nil {
		return nil, errors.Errorf("Saga instance `%s` not found", sagaId)
	}

	return sagaInstance, nil
}
