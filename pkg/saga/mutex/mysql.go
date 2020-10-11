package mutex

import (
	"context"
	"database/sql"
	"github.com/pkg/errors"
	"time"
)

type MysqlMutex struct {
	db *sql.DB
}

func NewSqlMutex(db *sql.DB) Mutex {
	return &MysqlMutex{db: db}
}

func (m MysqlMutex) Lock(ctx context.Context, sagaId string) error {

	r := sql.NullInt64{}
	if err := m.db.QueryRowContext(ctx, "SELECT GET_LOCK(?, -1)", sagaId).Scan(&r); err != nil {
		return WithMutexErr(errors.Wrapf(err, "error acquiring lock for saga %s", sagaId))
	}
	/*
		Returns 1 if the lock was obtained successfully,
		0 if the attempt timed out (for example, because another client has previously locked the name),
		or NULL if an error occurred (such as running out of memory or the thread was killed with mysqladmin kill).
	*/

	if r.Int64 == 1 {
		return nil
	}

	return WithMutexErr(errors.Errorf("Got 0 when acquiring lock for saga %s. Request timed out.", sagaId))
}

func (m MysqlMutex) Release(ctx context.Context, sagaId string) error {
	r := sql.NullInt64{}
	ctx, _ = context.WithTimeout(ctx, time.Second * 5)
	if err := m.db.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", sagaId).Scan(&r); err != nil {
		return WithMutexErr(errors.Errorf("Got 0 when releasing lock for saga %s", sagaId))
	}

	return nil
}
