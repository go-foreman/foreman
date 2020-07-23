package mutex

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
)

type MutexErr struct {
	error
}

func WithMutexErr(err error) error {
	return MutexErr{err}
}

type Mutex interface {
	Lock(ctx context.Context, sagaId string) error
	Release(ctx context.Context, sagaId string) error
}


