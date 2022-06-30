package mutex

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-foreman/foreman/log"

	"github.com/go-foreman/foreman/saga"
	sagaSql "github.com/go-foreman/foreman/saga/sql"

	"github.com/pkg/errors"
)

type sqlLock struct {
	releaseFunc func(context.Context) error
}

func (l *sqlLock) Release(ctx context.Context) error {
	return l.releaseFunc(ctx)
}

type mysqlMutex struct {
	db     *sagaSql.DB
	logger log.Logger
}

func NewSqlMutex(db *sagaSql.DB, driver saga.SQLDriver, logger log.Logger) Mutex {
	if driver == saga.MYSQLDriver {
		return &mysqlMutex{db: db, logger: logger}
	}
	return &pgsqlMutex{db: db, logger: logger}
}

func (m *mysqlMutex) Lock(ctx context.Context, sagaId string) (Lock, error) {
	conn, err := m.db.Conn(ctx, sagaId, true)

	if err != nil {
		return nil, WithMutexErr(errors.Wrapf(err, "obtaining a connection from pool for saga %s", sagaId))
	}

	r := sql.NullInt64{}
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, -1);", sagaId).Scan(&r); err != nil {
		//@todo I'm not so sure that connection needs to be closed in case of failure, what about retries?
		// NOTE: Multiple replicas of messagebus do not guarantee that retry will be handle by this specific instance.
		closingErr := conn.Close(true)
		return nil, WithMutexErr(errors.Wrapf(err, "acquiring lock for saga %s. %s", sagaId, closingErr))
	}

	/*
		Returns 1 if the lock was obtained successfully,
		0 if the attempt timed out (for example, because another client has previously locked the name),
		or NULL if an error occurred (such as running out of memory or the thread was killed with mysqladmin kill).
	*/
	if r.Int64 == 1 {
		return &sqlLock{
			releaseFunc: func(ctx context.Context) error {
				return m.release(ctx, conn, sagaId)
			},
		}, nil
	}

	closingErr := conn.Close(true)

	return nil, WithMutexErr(errors.Errorf("got error status %d when acquiring lock for saga %s. %s", r.Int64, sagaId, closingErr))
}

func (m *mysqlMutex) release(ctx context.Context, conn *sagaSql.Conn, sagaId string) error {
	r := sql.NullInt64{}
	if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?);", sagaId).Scan(&r); err != nil {
		closingErr := conn.Close(true)
		return WithMutexErr(errors.Wrapf(err, "releasing lock for saga %s. %s", sagaId, closingErr))
	}

	// Returns 1 if the lock was released, 0 if the lock was not established by this thread (in which case the lock is not released),
	// and NULL if the named lock did not exist. The lock does not exist if it was never obtained by a call to GET_LOCK() or if it has previously been released.
	if r.Int64 != 1 || !r.Valid {
		closingErr := conn.Close(true)
		return WithMutexErr(errors.Errorf("lock was not established by this thread for saga %s. %s", sagaId, closingErr))
	}

	if err := conn.Close(true); err != nil {
		return WithMutexErr(errors.Wrapf(err, "closing connection for saga's %s mutex", sagaId))
	}

	return nil
}

type pgsqlMutex struct {
	db     *sagaSql.DB
	logger log.Logger
}

func (p *pgsqlMutex) Lock(ctx context.Context, sagaId string) (Lock, error) {
	var (
		conn *sagaSql.Conn
		err  error
	)

	retries := 3

	// this retries are needed because database/sql with pg for some reason returns a connection which is closed already
	// test "release not existing lock" failed right after "failed to acquire a lock".
	// the last one closes connection on fail and for some reason this connection was assigned to the first test,
	// thus failing with message: ---"acquiring lock for saga bbb. sql: connection is already closed. also failed to close connection sql: connection is already closed"---
	// https://github.com/golang/go/issues/39449
	// https://github.com/golang/go/issues/32530

	// I'll create an issue and try to investigate into this bug
	for i := 0; i < retries; i++ {
		conn, err = p.db.Conn(ctx, sagaId, true)

		if err != nil {
			return nil, WithMutexErr(errors.Wrapf(err, "obtaining a connection from pool for saga %s", sagaId))
		}

		if err := conn.PingContext(ctx); err != nil {
			p.logger.Logf(log.ErrorLevel, "locking saga '%s'. error verifying that obtained mutex connection is alive. %s", sagaId, err)
			if i < retries-1 {
				if err := conn.Close(true); err != nil {
					p.logger.Logf(log.ErrorLevel, "locking '%s'. closing connection after verifying that it's dead %s", sagaId, err)
				}
				continue
			}
		}

		break
	}

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext($1));`, sagaId); err != nil {
		errMsg := fmt.Sprintf("acquiring lock for saga %s. %s", sagaId, err)

		if closingErr := conn.Close(true); closingErr != nil {
			errMsg = fmt.Sprintf("%s. also failed to close connection %s", errMsg, closingErr.Error())
		}
		return nil, WithMutexErr(errors.New(errMsg))
	}

	return &sqlLock{
		releaseFunc: func(ctx context.Context) error {
			return p.release(ctx, conn, sagaId)
		},
	}, nil
}

func (p *pgsqlMutex) release(ctx context.Context, conn *sagaSql.Conn, sagaId string) error {
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext($1));", sagaId); err != nil {
		closingErr := conn.Close(true)
		return WithMutexErr(errors.Wrapf(err, "releasing lock for saga %s. %s", sagaId, closingErr))
	}

	if err := conn.Close(true); err != nil {
		return WithMutexErr(errors.Wrapf(err, "closing mutex connection of saga %s", sagaId))
	}

	return nil
}
