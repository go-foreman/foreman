package mutex

import (
	"context"
	"database/sql"
)

type pgsqlMutex struct {
	db *sql.DB
}

func (p pgsqlMutex) Lock(ctx context.Context, sagaId string) error {
	panic("implement me")
}

func (p pgsqlMutex) Release(ctx context.Context, sagaId string) error {
	panic("implement me")
}

func NewPgsqlMutex(db *sql.DB) Mutex {
	return &pgsqlMutex{db: db}
}
