package status

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-foreman/foreman/log"
	"github.com/go-foreman/foreman/saga"
	"github.com/pkg/errors"
)

type StatusResponse struct {
	SagaUID string      `json:"saga_uid"`
	Status  string      `json:"status"`
	Payload interface{} `json:"payload"`
	Events  []SagaEvent `json:"events"`
}

type SagaEvent struct {
	saga.HistoryEvent
}

//go:generate mockgen --build_flags=--mod=mod -destination ./mock_test.go -package status . StatusService

type StatusService interface {
	GetStatus(ctx context.Context, sagaId string) (*StatusResponse, error)
	GetFilteredBy(ctx context.Context, sagaId, status, sagaType string) ([]StatusResponse, error)
}

func NewStatusService(store saga.Store) StatusService {
	return &statusService{sagaStore: store}
}

type statusService struct {
	sagaStore saga.Store
}

func (s statusService) GetStatus(ctx context.Context, sagaId string) (*StatusResponse, error) {
	sagaInstance, err := s.sagaStore.GetById(ctx, sagaId)

	if err != nil {
		return nil, errors.Wrapf(err, "error loading saga '%s'", sagaId)
	}

	if sagaInstance == nil {
		return nil, NewResponseError(http.StatusNotFound, errors.Errorf("saga '%s' not found", sagaId))
	}

	events := make([]SagaEvent, len(sagaInstance.HistoryEvents()))

	for i, ev := range sagaInstance.HistoryEvents() {
		events[i] = SagaEvent{ev}
	}

	return &StatusResponse{SagaUID: sagaId, Status: sagaInstance.Status().String(), Payload: sagaInstance.Saga(), Events: events}, nil
}

func (s statusService) GetFilteredBy(ctx context.Context, sagaId, status, sagaName string) ([]StatusResponse, error) {

	var opts []saga.FilterOption

	if sagaId != "" {
		opts = append(opts, saga.WithSagaId(sagaId))
	}

	if status != "" {
		opts = append(opts, saga.WithStatus(status))
	}

	if sagaName != "" {
		opts = append(opts, saga.WithSagaName(sagaName))
	}

	if len(opts) == 0 {
		return nil, NewResponseError(http.StatusBadRequest, errors.New("no filters specified"))
	}

	sagas, err := s.sagaStore.GetByFilter(ctx, opts...)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	resp := make([]StatusResponse, len(sagas))

	for i, instance := range sagas {
		events := make([]SagaEvent, len(instance.HistoryEvents()))

		for j, ev := range instance.HistoryEvents() {
			events[j] = SagaEvent{ev}
		}

		resp[i] = StatusResponse{
			SagaUID: instance.UID(),
			Status:  instance.Status().String(),
			Payload: instance.Saga(),
			Events:  events,
		}
	}

	return resp, nil
}

type StatusHandler struct {
	service StatusService
	logger  log.Logger
}

func NewStatusHandler(logger log.Logger, service StatusService) *StatusHandler {
	return &StatusHandler{service: service, logger: logger}
}

func (h *StatusHandler) GetStatus(resp http.ResponseWriter, r *http.Request) {

	sagaId := strings.TrimPrefix(r.URL.Path, "/sagas/")

	if sagaId == "" {
		resp.WriteHeader(http.StatusBadRequest)

		if _, err := resp.Write([]byte("Saga id is empty")); err != nil {
			h.logger.Log(log.ErrorLevel, err)
		}

		return
	}

	statusResp, err := h.service.GetStatus(r.Context(), sagaId)

	if err != nil {
		h.logger.Log(log.ErrorLevel, err)

		if respErr, ok := err.(ResponseError); ok {
			resp.WriteHeader(respErr.Status())
		} else {
			resp.WriteHeader(http.StatusInternalServerError)
		}

		if _, err := resp.Write([]byte(err.Error())); err != nil {
			h.logger.Log(log.ErrorLevel, err)
		}
		return
	}

	status, err := json.Marshal(statusResp)

	if err != nil {
		h.logger.Log(log.ErrorLevel, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")

	if _, err := resp.Write(status); err != nil {
		h.logger.Log(log.ErrorLevel, err)
	}
}

func (h *StatusHandler) GetFilteredBy(resp http.ResponseWriter, r *http.Request) {

	sagaId := r.URL.Query().Get("sagaId")
	status := r.URL.Query().Get("status")
	sagaType := r.URL.Query().Get("sagaType")

	statusesResp, err := h.service.GetFilteredBy(r.Context(), sagaId, status, sagaType)

	if err != nil {
		h.logger.Log(log.ErrorLevel, err)

		if respErr, ok := err.(ResponseError); ok {
			resp.WriteHeader(respErr.Status())
		} else {
			resp.WriteHeader(http.StatusInternalServerError)
		}

		if _, err := resp.Write([]byte(err.Error())); err != nil {
			h.logger.Log(log.ErrorLevel, err)
		}
		return
	}

	rawResponse, err := json.Marshal(statusesResp)

	if err != nil {
		h.logger.Log(log.ErrorLevel, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json")

	if _, err := resp.Write(rawResponse); err != nil {
		h.logger.Log(log.ErrorLevel, err)
	}
}

type ResponseError struct {
	error
	status int
}

//Status returns http status code
func (e ResponseError) Status() int {
	return e.status
}

func NewResponseError(status int, err error) ResponseError {
	return ResponseError{status: status, error: err}
}
