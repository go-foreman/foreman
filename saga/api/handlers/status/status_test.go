package status

import (
	"context"
	"testing"

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
			assert.EqualError(t, err, "saga '123' not found")
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
			assert.EqualError(t, err, "no filters specified")
		})
	})
}

type dataContract struct {
	message.ObjectMeta
}
