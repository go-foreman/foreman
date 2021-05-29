package mutex

import (
	"context"
	"testing"
	"time"

	"github.com/go-foreman/foreman/saga"
	"github.com/go-foreman/foreman/saga/mutex"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		id := "555"

		assert.NoError(t, pgMutex.Lock(ctx, id))
		assert.NoError(t, pgMutex.Release(ctx, id))

		_, err := m.Connection().ExecContext(ctx, "SELECT pg_advisory_unlock($1);", id)
		assert.NoError(t, err)
	})
}

func (m *pgMutexTest) createMutexService() mutex.Mutex {
	return mutex.NewSqlMutex(m.Connection(), saga.PGDriver)
}
