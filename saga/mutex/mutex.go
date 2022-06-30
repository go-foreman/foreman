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

//go:generate mockgen --build_flags=--mod=mod -destination ./../../testing/mocks/saga/mutex/mutex.go -package mutex . Mutex,Lock

type Mutex interface {
	Lock(ctx context.Context, sagaId string) (Lock, error)
}
