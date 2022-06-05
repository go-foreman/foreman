package mutex

import (
	"context"
)

type MutexErr struct {
	error
}

func WithMutexErr(err error) error {
	return MutexErr{err}
}

type Lock interface {
	Release(ctx context.Context) error
}

type Mutex interface {
	Lock(ctx context.Context, sagaId string) (Lock, error)
}
