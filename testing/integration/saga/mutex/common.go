package mutex

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-foreman/foreman/saga/mutex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func testSQLMutexUseCases(t *testing.T, mutexFabric func() mutex.Mutex, dbConnection *sql.DB) {
	sqlMutex := mutexFabric()

	t.Run("acquire and release a mutex sequentially", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
		defer cancel()

		id := "xxx"

		require.NoError(t, sqlMutex.Lock(ctx, id))
		assert.NoError(t, sqlMutex.Release(ctx, id))

		require.NoError(t, sqlMutex.Lock(ctx, id))
		assert.NoError(t, sqlMutex.Release(ctx, id))
	})

	t.Run("wait to acquire locked mutex", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 1)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "yyy"
		assert.NoError(t, sqlMutex.Lock(ctx, id))

		go func() {
			time.Sleep(time.Millisecond * 100)
			assert.NoError(t, sqlMutex.Release(ctx, id))
		}()

		//this goroutine should wait until first lock is release
		go func() {
			time.Sleep(time.Millisecond * 150)
			waitingCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 200)
			defer cancel()

			assert.NoError(t, sqlMutex.Lock(waitingCtx, id))
			select {
			case <- ctx.Done():
				return
			case releaseCh <- struct{}{}:
				return
			}
		}()

		select {
		case <- ctx.Done():
			t.FailNow()
		case <- releaseCh:
		}
	})

	t.Run("wait to acquire locked mutex from another service instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 100)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "zzz"
		assert.NoError(t, sqlMutex.Lock(ctx, id))

		go func() {
			time.Sleep(time.Millisecond * 100)
			assert.NoError(t, sqlMutex.Release(ctx, id))
		}()

		//this goroutine should wait until first lock is release
		go func() {
			waitingCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 200)
			defer cancel()
			anotherInstanceMutex := mutexFabric()
			assert.NoError(t, anotherInstanceMutex.Lock(waitingCtx, id))
			select {
			case <- ctx.Done():
				return
			case releaseCh <- struct{}{}:
				return
			}
		}()

		select {
		case <- ctx.Done():
			t.FailNow()
		case <- releaseCh:
		}
	})

	t.Run("failed to acquire a lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "aaa"
		assert.NoError(t, sqlMutex.Lock(ctx, id))

		//this goroutine should wait until first lock is release
		go func() {
			waitingCtx, cancel := context.WithTimeout(ctx, time.Millisecond * 200)
			defer cancel()

			err := sqlMutex.Lock(waitingCtx, id)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), fmt.Sprintf("acquiring lock for saga %s", id))
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

		assert.NoError(t, sqlMutex.Release(ctx, id))
	})

	t.Run("release not existing lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
		defer cancel()

		id := "bbb"

		require.NoError(t, sqlMutex.Lock(ctx, id))
		assert.NoError(t, sqlMutex.Release(ctx, id))

		err := sqlMutex.Release(ctx, id)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection which acquiring lock is not found in runtime map. Was Release() called after processing a message?")
	})
}
