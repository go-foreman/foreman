package saga

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-foreman/foreman/saga/mutex"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
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

	mysqlMutex := mutex.NewMysqlSqlMutex(m.Connection())

	t.Run("acquire and release a mutex sequentially", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
		defer cancel()

		id := "111"

		assert.NoError(t, mysqlMutex.Lock(ctx, id))
		assert.NoError(t, mysqlMutex.Release(ctx, id))

		assert.NoError(t, mysqlMutex.Lock(ctx, id))
		assert.NoError(t, mysqlMutex.Release(ctx, id))
	})

	t.Run("wait to acquire locked id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 1)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "222"
		assert.NoError(t, mysqlMutex.Lock(ctx, id))

		go func() {
			time.Sleep(time.Millisecond * 100)
			assert.NoError(t, mysqlMutex.Release(ctx, id))
		}()

		//this goroutine should wait until first lock is release
		go func() {
			waitingCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 200)
			defer cancel()

			assert.NoError(t, mysqlMutex.Lock(waitingCtx, id))
			select {
			case <- ctx.Done():
				return
			case releaseCh <- struct{}{}:
				return
			}
		}()

		select {
		case <- time.After(time.Millisecond * 500):
			t.FailNow()
		case <- releaseCh:
		}
	})

	t.Run("failed to acquire a lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 1)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "333"
		assert.NoError(t, mysqlMutex.Lock(ctx, id))

		//this goroutine should wait until first lock is release
		go func() {
			waitingCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 200)
			defer cancel()

			err := mysqlMutex.Lock(waitingCtx, id)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("error acquiring lock for saga %s: context deadline exceeded", id))
			select {
			case <- ctx.Done():
				return
			case releaseCh <- struct{}{}:
				return
			}
		}()

		select {
		case <- time.After(time.Millisecond * 500):
			t.FailNow()
		case <- releaseCh:
		}
	})

	t.Run("release not existing lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
		defer cancel()

		id := "444"

		assert.NoError(t, mysqlMutex.Lock(ctx, id))
		assert.NoError(t, mysqlMutex.Release(ctx, id))

		err := mysqlMutex.Release(ctx, id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection which acquiring lock is not found in runtime map. Was Release() called after processing a message?")
	})

	t.Run("manually fail to release already released lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
		defer cancel()

		id := "555"

		assert.NoError(t, mysqlMutex.Lock(ctx, id))
		assert.NoError(t, mysqlMutex.Release(ctx, id))

		r := sql.NullInt64{}
		assert.NoError(t, m.Connection().QueryRowContext(ctx, "SELECT RELEASE_LOCK(?);", id).Scan(&r))

		assert.Equal(t, int64(0), r.Int64)
	})
}
