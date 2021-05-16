package mutex

import (
	"context"
	"github.com/go-foreman/foreman/saga"
	"github.com/go-foreman/foreman/saga/mutex"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type pgMutexTest struct {
	intSuite.PgSuite
}

func TestPgMutexSuite(t *testing.T) {
	pgMutexTest := &pgMutexTest{}
	suite.Run(t, pgMutexTest)
}

func (m *pgMutexTest) TestMutexStore() {
	t := m.T()

	pgMutex := m.createMutexService()

	testSQLMutexUseCases(t, m.createMutexService, m.Connection())

	t.Run("manually fail to release already released lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
		defer cancel()

		id := "555"

		assert.NoError(t, pgMutex.Lock(ctx, id))
		assert.NoError(t, pgMutex.Release(ctx, id))

		_, err := m.Connection().ExecContext(ctx, "SELECT pg_advisory_unlock($1);", id)
		assert.NoError(t, err)
	})
}

func (m *pgMutexTest) TestSqlProblem() {
	db := m.Connection()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 60)
	defer cancel()

	conn, err := db.Conn(ctx)
	require.NoError(m.T(), err)
	require.NoError(m.T(), conn.PingContext(ctx))

	go func() {
		time.Sleep(time.Millisecond)
		require.NoError(m.T(), conn.Close())
	}()

	anotherConn, err := db.Conn(ctx)
	require.NoError(m.T(), err)
	require.NoError(m.T(), anotherConn.PingContext(ctx))

	go func() {
		time.Sleep(time.Millisecond)
		require.NoError(m.T(), anotherConn.Close())
	}()

	another2Conn, err := db.Conn(ctx)
	require.NoError(m.T(), err)
	require.NoError(m.T(), another2Conn.PingContext(ctx))

	go func() {
		time.Sleep(time.Millisecond)
		require.NoError(m.T(), another2Conn.Close())
	}()

	time.Sleep(time.Millisecond * 10)

	heh, err := db.Conn(ctx)
	require.NoError(m.T(), err)
	require.NoError(m.T(), heh.PingContext(ctx))
}

func (m *pgMutexTest) createMutexService() mutex.Mutex {
	return mutex.NewSqlMutex(m.Connection(), saga.PGDriver)
}
