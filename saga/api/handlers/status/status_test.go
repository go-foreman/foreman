package status

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-foreman/foreman/testing/log"
	"github.com/stretchr/testify/require"

	"github.com/go-foreman/foreman/pubsub/message"

	"github.com/pkg/errors"

	"github.com/go-foreman/foreman/saga"

	sagaMock "github.com/go-foreman/foreman/testing/mocks/saga"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestStatusService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storeMock := sagaMock.NewMockStore(ctrl)

	statusService := NewStatusService(storeMock)

	t.Run("getStatus", func(t *testing.T) {
		t.Run("no error", func(t *testing.T) {
			ctx := context.Background()
			sagaId := "123"

			sagaExample := sagaMock.NewMockSaga(ctrl)
			sagaInstance := saga.NewSagaInstance(sagaId, "", sagaExample)
			sagaInstance.AddHistoryEvent(&dataContract{}, nil)

			storeMock.
				EXPECT().
				GetById(ctx, sagaId).
				Return(sagaInstance, nil)

			resp, err := statusService.GetStatus(ctx, sagaId)
			assert.NoError(t, err)
			assert.Equal(t, resp.SagaUID, sagaId)
			assert.Equal(t, resp.Status, "created")
			assert.Equal(t, resp.Payload, sagaExample)
			assert.Equal(t, resp.Events, []SagaEvent{{sagaInstance.HistoryEvents()[0]}})
		})

		t.Run("error loading saga by id", func(t *testing.T) {
			ctx := context.Background()
			sagaId := "123"

			storeMock.
				EXPECT().
				GetById(ctx, sagaId).
				Return(nil, errors.New("some error"))

			_, err := statusService.GetStatus(ctx, sagaId)
			assert.Error(t, err)
			assert.EqualError(t, err, "error loading saga '123': some error")
		})

		t.Run("saga id not found", func(t *testing.T) {
			ctx := context.Background()
			sagaId := "123"

			storeMock.
				EXPECT().
				GetById(ctx, sagaId).
				Return(nil, nil)

			_, err := statusService.GetStatus(ctx, sagaId)
			assert.Error(t, err)
			respErr, ok := err.(ResponseError)
			require.True(t, ok)

			assert.Equal(t, respErr.Error(), "saga '123' not found")
			assert.Equal(t, respErr.Status(), http.StatusNotFound)
		})
	})

	t.Run("get filtered by", func(t *testing.T) {
		t.Run("no error", func(t *testing.T) {
			ctx := context.Background()
			sagaId := "123"

			sagaExample := sagaMock.NewMockSaga(ctrl)
			sagaInstance := saga.NewSagaInstance(sagaId, "", sagaExample)
			sagaInstance.AddHistoryEvent(&dataContract{}, nil)

			storeMock.
				EXPECT().
				GetByFilter(ctx, gomock.Any()).
				Do(func(ctx context.Context, filters ...saga.FilterOption) {
					assert.Len(t, filters, 3)
				}).
				Return([]saga.Instance{sagaInstance}, nil)

			resp, err := statusService.GetFilteredBy(ctx, sagaId, "in_progress", "someSagaType")
			assert.NoError(t, err)

			assert.Len(t, resp, 1)
			assert.Equal(t, resp[0].SagaUID, sagaId)
			assert.Equal(t, resp[0].Status, "created")
			assert.Equal(t, resp[0].Payload, sagaExample)
			assert.Equal(t, resp[0].Events, []SagaEvent{{sagaInstance.HistoryEvents()[0]}})
		})

		t.Run("error filtering", func(t *testing.T) {
			ctx := context.Background()
			sagaId := "123"

			storeMock.
				EXPECT().
				GetByFilter(ctx, gomock.Any()).
				Do(func(ctx context.Context, filters ...saga.FilterOption) {
					assert.Len(t, filters, 3)
				}).
				Return(nil, errors.New("some error"))

			resp, err := statusService.GetFilteredBy(ctx, sagaId, "in_progress", "someSagaType")
			assert.Error(t, err)
			assert.EqualError(t, err, "some error")
			assert.Nil(t, resp)
		})

		t.Run("no filters specified", func(t *testing.T) {
			_, err := statusService.GetFilteredBy(context.Background(), "", "", "")
			assert.Error(t, err)

			respErr, ok := err.(ResponseError)
			require.True(t, ok)

			assert.Equal(t, respErr.Error(), "no filters specified")
			assert.Equal(t, respErr.Status(), http.StatusBadRequest)
		})
	})
}

type dataContract struct {
	message.ObjectMeta
}

func TestHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	statusServiceMock := NewMockStatusService(ctrl)
	testLogger := log.NewNilLogger()

	handler := NewStatusHandler(testLogger, statusServiceMock)

	t.Run("Status", func(t *testing.T) {
		t.Run("sagaid is empty", func(t *testing.T) {

			req, err := http.NewRequest("GET", "http://localhost:8000/sagas/", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.GetStatus(rr, req)

			assert.Contains(t, rr.Body.String(), "Saga id is empty")
		})

		t.Run("get Status", func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://localhost:8000/sagas/123", nil)
			require.NoError(t, err)

			statusResp := &StatusResponse{
				SagaUID: "123",
				Status:  "in_progress",
			}

			statusServiceMock.
				EXPECT().
				GetStatus(req.Context(), "123").
				Return(statusResp, nil)

			rr := httptest.NewRecorder()
			handler.GetStatus(rr, req)

			statusRespMarshalled, _ := json.Marshal(statusResp)
			assert.Contains(t, rr.Body.String(), string(statusRespMarshalled))
			assert.Equal(t, rr.Header().Get("Content-Type"), "application/json")
		})

		t.Run("service returns an error", func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://localhost:8000/sagas/123", nil)
			require.NoError(t, err)

			statusServiceMock.
				EXPECT().
				GetStatus(req.Context(), "123").
				Return(nil, errors.New("some error"))

			rr := httptest.NewRecorder()
			handler.GetStatus(rr, req)

			assert.Equal(t, rr.Code, http.StatusInternalServerError)
			assert.Contains(t, rr.Body.String(), "some error")
		})

		t.Run("service returns Status error", func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://localhost:8000/sagas/123", nil)
			require.NoError(t, err)

			statusServiceMock.
				EXPECT().
				GetStatus(req.Context(), "123").
				Return(nil, NewResponseError(409, errors.New("some error")))

			rr := httptest.NewRecorder()
			handler.GetStatus(rr, req)

			assert.Equal(t, rr.Code, http.StatusConflict)
			assert.Contains(t, rr.Body.String(), "some error")
		})
	})

	t.Run("get by filter", func(t *testing.T) {
		t.Run("all filters", func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://localhost:8000/sagas?&Status=created&SagaName=someType", nil)
			require.NoError(t, err)

			statuses := []StatusResponse{
				{
					SagaUID: "123",
					Status:  "created",
				},
				{
					SagaUID: "111",
					Status:  "created",
				},
			}

			statusServiceMock.
				EXPECT().
				GetFilteredBy(req.Context(), "", "created", "someType").
				Return(statuses, nil)

			rr := httptest.NewRecorder()
			handler.GetFilteredBy(rr, req)

			statusRespMarshalled, _ := json.Marshal(statuses)
			assert.Contains(t, rr.Body.String(), string(statusRespMarshalled))
			assert.Equal(t, rr.Header().Get("Content-Type"), "application/json")
		})

		t.Run("get filtered returns a response error", func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://localhost:8000/sagas?&Status=created&SagaName=someType", nil)
			require.NoError(t, err)

			statusServiceMock.
				EXPECT().
				GetFilteredBy(req.Context(), "", "created", "someType").
				Return(nil, NewResponseError(http.StatusBadRequest, errors.New("some error")))

			rr := httptest.NewRecorder()
			handler.GetFilteredBy(rr, req)

			assert.Equal(t, rr.Code, http.StatusBadRequest)
			assert.Contains(t, rr.Body.String(), "some error")
		})

		t.Run("get filtered returns an error", func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://localhost:8000/sagas?&Status=created&SagaName=someType", nil)
			require.NoError(t, err)

			statusServiceMock.
				EXPECT().
				GetFilteredBy(req.Context(), "", "created", "someType").
				Return(nil, errors.New("some error"))

			rr := httptest.NewRecorder()
			handler.GetFilteredBy(rr, req)

			assert.Equal(t, rr.Code, http.StatusInternalServerError)
			assert.Contains(t, rr.Body.String(), "some error")
		})
	})
}
