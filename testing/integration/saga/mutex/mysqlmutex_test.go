package mutex

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-foreman/foreman/testing/log"

	sagaSql "github.com/go-foreman/foreman/saga/sql"

	"github.com/go-foreman/foreman/saga"
	"github.com/go-foreman/foreman/saga/mutex"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type mysqlMutexTest struct {
	intSuite.MysqlSuite
}

func TestMysqlMutexSuite(t *testing.T) {
	mysqlMutexTest := &mysqlMutexTest{}
	suite.Run(t, mysqlMutexTest)
}

func (m *mysqlMutexTest) TestMysqlMutexStore() {
	t := m.T()

	mysqlMutex := m.createMutexService()

	testSQLMutexUseCases(t, m.createMutexService, m.Connection())

	t.Run("manually fail to release already released lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		id := "555"

		lock, err := mysqlMutex.Lock(ctx, id)
		assert.NoError(t, err)
		assert.NoError(t, lock.Release(ctx))

		r := sql.NullInt64{}
		assert.NoError(t, m.Connection().QueryRowContext(ctx, "SELECT RELEASE_LOCK(?);", id).Scan(&r))

		assert.Equal(t, int64(0), r.Int64)
	})
}

func (m *mysqlMutexTest) createMutexService() mutex.Mutex {
	return mutex.NewSqlMutex(sagaSql.NewDB(m.Connection()), saga.MYSQLDriver, log.NewNilLogger())
}
