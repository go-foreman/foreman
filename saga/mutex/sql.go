package mutex

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-foreman/foreman/saga"
	"sync"

	"github.com/pkg/errors"
)

type mysqlMutex struct {
	db          *sql.DB
	mapLock     sync.Mutex
	connections map[string]*sql.Conn
}

func NewSqlMutex(db *sql.DB, driver saga.SQLDriver) Mutex {
	if driver == saga.MYSQLDriver {
		return &mysqlMutex{db: db, connections: make(map[string]*sql.Conn)}
	}
	return &pgsqlMutex{db: db, connections: make(map[string]*sql.Conn)}
}

func (m *mysqlMutex) Lock(ctx context.Context, sagaId string) error {
	conn, err := m.db.Conn(ctx)

	if err != nil {
		return WithMutexErr(errors.Wrapf(err, "obtaining a connection from pool for saga %s", sagaId))
	}

	r := sql.NullInt64{}
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, -1);", sagaId).Scan(&r); err != nil {
		closingErr := conn.Close()
		return WithMutexErr(errors.Wrapf(err, "acquiring lock for saga %s. %s", sagaId, closingErr))
	}

	/*
		Returns 1 if the lock was obtained successfully,
		0 if the attempt timed out (for example, because another client has previously locked the name),
		or NULL if an error occurred (such as running out of memory or the thread was killed with mysqladmin kill).
	*/
	if r.Int64 == 1 {
		//we lock map here because GET_LOCK allows us to acquire a lock, other clients won't be able to pass that point.
		m.mapLock.Lock()
		defer m.mapLock.Unlock()

		m.connections[sagaId] = conn

		return nil
	}

	closingErr := conn.Close()

	return WithMutexErr(errors.Errorf("Got 0 when acquiring lock for saga %s. %s", sagaId, closingErr))
}

func (m *mysqlMutex) Release(ctx context.Context, sagaId string) error {
	m.mapLock.Lock()
	conn, exists := m.connections[sagaId]
	if !exists {
		m.mapLock.Unlock()
		return WithMutexErr(errors.Errorf("connection which acquiring lock is not found in runtime map. Was Release() called after processing a message?"))
	}

	r := sql.NullInt64{}
	if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?);", sagaId).Scan(&r); err != nil {
		closingErr := conn.Close()
		return WithMutexErr(errors.Wrapf(err, "releasing lock for saga %s. %s", sagaId, closingErr))
	}

	if r.Int64 != 1 {
		closingErr := conn.Close()
		return WithMutexErr(errors.Errorf("lock was not established by this thread for saga %s. %s", sagaId, closingErr))
	}

	delete(m.connections, sagaId)
	m.mapLock.Unlock()

	if err := conn.Close(); err != nil {
		return WithMutexErr(errors.Wrapf(err, "closing connection for saga's %s mutex", sagaId))
	}

	return nil
}

type pgsqlMutex struct {
	db *sql.DB
	mapLock     sync.Mutex
	connections map[string]*sql.Conn
}

func (p *pgsqlMutex) Lock(ctx context.Context, sagaId string) error {
	var (
		conn *sql.Conn
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
		conn, err = p.db.Conn(ctx)

		if err != nil {
			return WithMutexErr(errors.Wrapf(err, "obtaining a connection from pool for saga %s", sagaId))
		}

		if err := conn.PingContext(ctx); err != nil {
			if i < retries -1 {
				continue
			}
		}

		break
	}

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext($1));`, sagaId); err != nil {
		errMsg := fmt.Sprintf("acquiring lock for saga %s. %s", sagaId, err)

		if closingErr := conn.Close(); closingErr != nil {
			errMsg = fmt.Sprintf("%s. also failed to close connection %s", errMsg, closingErr.Error())
		}
		return WithMutexErr(errors.New(errMsg))
	}

	p.mapLock.Lock()
	defer p.mapLock.Unlock()

	p.connections[sagaId] = conn

	return nil
}

func (p *pgsqlMutex) Release(ctx context.Context, sagaId string) error {
	p.mapLock.Lock()
	defer p.mapLock.Unlock()

	conn, exists := p.connections[sagaId]
	if !exists {
		return WithMutexErr(errors.Errorf("connection which acquiring lock is not found in runtime map. Was Release() called after processing a message?"))
	}

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock(hashtext($1));", sagaId); err != nil {
		closingErr := conn.Close()
		return WithMutexErr(errors.Wrapf(err, "releasing lock for saga %s. %s", sagaId, closingErr))
	}

	delete(p.connections, sagaId)

	if err := conn.Close(); err != nil {
		return WithMutexErr(errors.Wrapf(err, "closing mutex connection of saga %s", sagaId))
	}

	return nil
}
