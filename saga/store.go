package saga

import (
	"context"
	"github.com/go-foreman/foreman/pubsub/message"
	"time"
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

type sagaModel struct {
	ID        string
	ParentID  string
	Name      string
	Payload   []byte
	Status    string
	StartedAt time.Time
	UpdatedAt time.Time
}

type historyEventModel struct {
	message.Metadata
	CreatedAt    time.Time
	Payload      []byte
	OriginSource string
	SagaStatus   string
	Description  string
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
