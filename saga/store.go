package saga

import (
	"context"
	"github.com/pkg/errors"
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

func WithSagaName(sagaName string) FilterOption {
	return func(opts *filterOptions) {
		opts.sagaName = sagaName
	}
}

type filterOptions struct {
	sagaId   string
	status   string
	sagaName string
}

func statusFromStr(str string) (status, error) {
	statuses := []status{sagaStatusInProgress, sagaStatusFailed, sagaStatusInProgress, sagaStatusCompensating, sagaStatusCompleted, sagaStatusCreated}
	for _, s := range statuses {
		if string(s) == str {
			return s, nil
		}
	}

	return "", errors.Errorf("unknown status string")
}