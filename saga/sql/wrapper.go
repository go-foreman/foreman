package sql

import (
	"context"
	"database/sql"
	"sync"

	"github.com/pkg/errors"
)

type DB struct {
	*sql.DB
	connectionsLock  *sync.RWMutex
	connectionsInUse map[string]*Conn
}

func NewDB(db *sql.DB) *DB {
	return &DB{
		DB:               db,
		connectionsLock:  &sync.RWMutex{},
		connectionsInUse: make(map[string]*Conn),
	}
}

func (m *DB) Conn(ctx context.Context, sagaUID string, lock bool) (*Conn, error) {
	m.connectionsLock.RLock()

	//check if connection for that sagaUID already exists
	wrappedConn, exists := m.connectionsInUse[sagaUID]

	m.connectionsLock.RUnlock()

	if exists {
		wrappedConn.registerClient()

		if lock {
			// try to lock that connection.
			if err := wrappedConn.acquireLock(ctx); err != nil {
				return nil, errors.Wrapf(err, "acquiring connection")
			}
		}

		// check if this connection is still alive. There might be cases when a connection wasn't released for some reason.
		if err := wrappedConn.PingContext(ctx); err == nil {
			return wrappedConn, nil
		}
	}

	conn, err := m.DB.Conn(ctx)

	if err != nil {
		return nil, errors.Wrapf(err, "obtaining a connection from pool for saga %s", sagaUID)
	}

	wrappedConn = &Conn{
		clientsMutex: &sync.Mutex{},
		clients:      1,
		lockingMutex: make(chan struct{}, 1),
		sagaUID:      sagaUID,
		Conn:         conn,
		releaseFunc:  m.releaseConnection,
	}

	if lock {
		wrappedConn.lockingMutex <- struct{}{}
	}

	m.connectionsLock.Lock()
	defer m.connectionsLock.Unlock()

	m.connectionsInUse[sagaUID] = wrappedConn

	return wrappedConn, nil
}

func (m *DB) releaseConnection(sagaUID string) {
	m.connectionsLock.Lock()
	defer m.connectionsLock.Unlock()

	delete(m.connectionsInUse, sagaUID)
}

type Conn struct {
	*sql.Conn
	clientsMutex *sync.Mutex
	clients      uint32
	lockingMutex chan struct{}
	sagaUID      string
	releaseFunc  func(sagaUID string)
}

func (c *Conn) registerClient() {
	c.clientsMutex.Lock()
	defer c.clientsMutex.Unlock()

	c.clients++
}

func (c *Conn) acquireLock(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return errors.New("context canceled while waiting for connection lock")
	case c.lockingMutex <- struct{}{}:
		return nil
	}
}

func (c *Conn) Close(unlock bool) error {
	c.clientsMutex.Lock()
	defer c.clientsMutex.Unlock()

	c.clients--

	if unlock {
		if len(c.lockingMutex) == 1 {
			<-c.lockingMutex
		} else {
			return errors.New("called conn.Close(true) on connection that wasn't locked")
		}
	}

	if c.clients == 0 {
		c.releaseFunc(c.sagaUID)
		return c.Conn.Close()
	}

	return nil
}
