package mutex

import (
	"context"
	"database/sql"
	"sync"

	"github.com/pkg/errors"
)

type MysqlMutex struct {
	mapLock     sync.Mutex
	db          *sql.DB
	connections map[string]*sql.Conn
}

func NewSqlMutex(db *sql.DB) Mutex {
	return &MysqlMutex{db: db, connections: make(map[string]*sql.Conn)}
}

func (m *MysqlMutex) Lock(ctx context.Context, sagaId string) error {
	conn, err := m.db.Conn(ctx)

	if err != nil {
		return WithMutexErr(errors.Wrapf(err, "error obtaining mutex mysql connection from pool for saga %s", sagaId))
	}

	r := sql.NullInt64{}
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, -1);", sagaId).Scan(&r); err != nil {
		conn.Close()
		return WithMutexErr(errors.Wrapf(err, "error acquiring lock for saga %s", sagaId))
	}

	m.mapLock.Lock()
	m.connections[sagaId] = conn
	m.mapLock.Unlock()

	/*
		Returns 1 if the lock was obtained successfully,
		0 if the attempt timed out (for example, because another client has previously locked the name),
		or NULL if an error occurred (such as running out of memory or the thread was killed with mysqladmin kill).
	*/

	if r.Int64 == 1 {
		return nil
	}

	conn.Close()
	return WithMutexErr(errors.Errorf("Got 0 when acquiring lock for saga %s", sagaId))
}

func (m *MysqlMutex) Release(ctx context.Context, sagaId string) error {
	m.mapLock.Lock()
	conn, exists := m.connections[sagaId]
	if !exists {
		return WithMutexErr(errors.Errorf("connection which acquiring lock is not found in runtime map. Was Release() called after processing a message?"))
	}
	defer conn.Close()

	delete(m.connections, sagaId)
	m.mapLock.Unlock()

	r := sql.NullInt64{}
	if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?);", sagaId).Scan(&r); err != nil {
		return WithMutexErr(errors.Errorf("error releasing lock for saga %s. %s", sagaId, err))
	}

	if r.Int64 == 1 {
		return nil
	}

	return WithMutexErr(errors.Errorf("lock was not established by this thread for saga %s", sagaId))
}
