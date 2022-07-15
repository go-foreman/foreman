package saga

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

const (
	sagaTableName        = "saga"
	sagaHistoryTableName = "saga_history"
)

type FilterOption func(opts *filterOptions)

//go:generate mockgen --build_flags=--mod=mod -destination ../testing/mocks/saga/store.go -package saga . Store

type InstancesBatch struct {
	Total int
	Items []Instance
}

type Store interface {
	Create(ctx context.Context, saga Instance) error
	GetById(ctx context.Context, sagaId string) (Instance, error)
	GetByFilter(ctx context.Context, filters ...FilterOption) (*InstancesBatch, error)
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

func WithOffsetAndLimit(offset int, limit int) FilterOption {
	return func(opts *filterOptions) {
		opts.offset = &offset
		opts.limit = &limit
	}
}

type filterOptions struct {
	sagaId   string
	status   string
	sagaName string
	limit    *int
	offset   *int
}

func statusFromStr(str string) (status, error) {
	statuses := []status{sagaStatusInProgress, sagaStatusFailed, sagaStatusInProgress, sagaStatusCompensating, sagaStatusCompleted, sagaStatusCreated, sagaStatusRecovering}
	for _, s := range statuses {
		if string(s) == str {
			return s, nil
		}
	}

	return "", errors.Errorf("unknown status string")
}

type sagaSqlModel struct {
	ID            sql.NullString
	ParentID      sql.NullString
	Name          sql.NullString
	Payload       []byte
	Status        sql.NullString
	LastFailedMsg []byte
	StartedAt     sql.NullTime
	UpdatedAt     sql.NullTime
}

type historyEventSqlModel struct {
	ID           sql.NullString
	SagaUID      sql.NullString
	Name         sql.NullString
	CreatedAt    sql.NullTime
	Payload      []byte
	OriginSource sql.NullString
	SagaStatus   sql.NullString
	TraceUID     sql.NullString
}
