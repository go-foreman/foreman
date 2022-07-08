package status

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
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

type Filters struct {
	SagaID   string
	SagaName string
	Status   string
	Offset   *int
	Limit    *int
}

type StatusService interface {
	GetStatus(ctx context.Context, sagaId string) (*StatusResponse, error)
	GetFilteredBy(ctx context.Context, filters *Filters) ([]StatusResponse, error)
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

func (s statusService) GetFilteredBy(ctx context.Context, filters *Filters) ([]StatusResponse, error) {

	var opts []saga.FilterOption

	if filters.SagaID != "" {
		opts = append(opts, saga.WithSagaId(filters.SagaID))
	}

	if filters.Status != "" {
		opts = append(opts, saga.WithStatus(filters.Status))
	}

	if filters.SagaName != "" {
		opts = append(opts, saga.WithSagaName(filters.SagaName))
	}

	if filters.Offset != nil {
		opts = append(opts, saga.WithOffset(*filters.Offset))
	}

	if filters.Limit != nil {
		opts = append(opts, saga.WithLimit(*filters.Limit))
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
		NewResponseWriterFromErrMsg("Saga id is empty", http.StatusBadRequest).write(resp, h.logger)
		return
	}

	statusResp, err := h.service.GetStatus(r.Context(), sagaId)

	if err != nil {
		NewResponseWriterFromError(err).write(resp, h.logger)
		return
	}

	NewResponseWriter(statusResp, http.StatusOK).write(resp, h.logger)
}

func (h *StatusHandler) GetFilteredBy(resp http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	sagaId := query.Get("sagaId")
	status := query.Get("status")
	sagaName := query.Get("sagaType")

	offset, err := h.getInt(query, "offset")

	if err != nil {
		NewResponseWriterFromError(err).write(resp, h.logger)
		return
	}

	limit, err := h.getInt(query, "limit")

	if err != nil {
		NewResponseWriterFromError(err).write(resp, h.logger)
		return
	}

	statusesResp, err := h.service.GetFilteredBy(r.Context(), &Filters{
		SagaID:   sagaId,
		SagaName: sagaName,
		Status:   status,
		Offset:   offset,
		Limit:    limit,
	})

	if err != nil {
		NewResponseWriterFromError(err).write(resp, h.logger)
		return
	}

	NewResponseWriter(statusesResp, http.StatusOK).write(resp, h.logger)
}

func (h *StatusHandler) getInt(values url.Values, paramName string) (*int, error) {
	paramValue := values.Get(paramName)
	if paramValue != "" {
		intValue, err := strconv.Atoi(paramValue)
		if err != nil {
			return nil, NewResponseError(http.StatusBadRequest, errors.Errorf("Query parameter '%s' is expected to be an integer", paramName))
		}

		return &intValue, nil
	}

	return nil, nil
}

type responseWriter struct {
	body   interface{}
	status int
}

func NewResponseWriterFromError(err error) *responseWriter {
	if respErr, ok := err.(ResponseError); ok {
		return &responseWriter{
			body:   respErr,
			status: respErr.Status(),
		}
	}

	return &responseWriter{
		body:   err,
		status: http.StatusInternalServerError,
	}
}

func NewResponseWriter(body interface{}, status int) *responseWriter {
	return &responseWriter{
		body:   body,
		status: status,
	}
}

func NewResponseWriterFromErrMsg(errMsg string, status int) *responseWriter {
	return NewResponseWriterFromError(NewResponseError(status, errors.New(errMsg)))
}

func (rw *responseWriter) encode() ([]byte, error) {
	var (
		respBody []byte
		err      error
	)

	if respErr, ok := rw.body.(error); ok {
		respBody = []byte(respErr.Error())
	} else {
		respBody, err = json.Marshal(rw.body)
	}

	return respBody, err
}

func (rw *responseWriter) write(resp http.ResponseWriter, logger log.Logger) {
	respBody, err := rw.encode()
	if err != nil {
		logger.Log(log.ErrorLevel, err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(rw.status)

	resp.Header().Set("Content-Type", "application/json")

	if _, err = resp.Write(respBody); err != nil {
		logger.Log(log.ErrorLevel, err)
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
