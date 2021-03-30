package saga

import (
	"context"
)

const (
	sagaTableName        = "saga"
	sagaHistoryTableName = "saga_history"
)

type FilterOption func(opts *filterOptions)

type Store interface {
	Create(ctx context.Context, saga Instance) error
	GetById(ctx context.Context, sagaId string) (Instance, error)
	GetByFilter(ctx context.Context, filters... FilterOption) ([]Instance, error)
	Update(ctx context.Context, saga Instance) error
	Delete(ctx context.Context, sagaId string) error
}

func WithSagaId(sagaId string) FilterOption {
	return func(opts *filterOptions) {
		opts.sagaId = sagaId
	}
}

func WithStatus(status string) FilterOption {
	return func(opts *filterOptions) {
		opts.status = status
	}
}

func WithSagaType(sagaType string) FilterOption {
	return func(opts *filterOptions) {
		opts.sagaType = sagaType
	}
}

type filterOptions struct {
	sagaId   string
	status   string
	sagaType string
}
