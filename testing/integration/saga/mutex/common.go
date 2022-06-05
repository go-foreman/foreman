package mutex

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/go-foreman/foreman/saga/mutex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSQLMutexUseCases(t *testing.T, mutexFabric func() mutex.Mutex, dbConnection *sql.DB) {
	sqlMutex := mutexFabric()

	t.Run("acquire and release a mutex sequentially", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		id := "xxx"

		lock, err := sqlMutex.Lock(ctx, id)

		require.NoError(t, err)
		assert.NoError(t, lock.Release(ctx))

		lock, err = sqlMutex.Lock(ctx, id)

		require.NoError(t, err)
		assert.NoError(t, lock.Release(ctx))
	})

	t.Run("wait to acquire locked mutex", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "yyy"
		lock, err := sqlMutex.Lock(ctx, id)
		assert.NoError(t, err)

		go func() {
			time.Sleep(time.Millisecond * 100)
			assert.NoError(t, lock.Release(ctx))
		}()

		//this goroutine should wait until first lock is release
		go func() {
			time.Sleep(time.Millisecond * 150)
			waitingCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
			defer cancel()

			_, err := sqlMutex.Lock(waitingCtx, id)
			assert.NoError(t, err)
			select {
			case <-ctx.Done():
				return
			case releaseCh <- struct{}{}:
				return
			}
		}()

		select {
		case <-ctx.Done():
			t.FailNow()
		case <-releaseCh:
		}
	})

	t.Run("wait to acquire locked mutex from another service instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "zzz"
		lock, err := sqlMutex.Lock(ctx, id)
		assert.NoError(t, err)

		go func() {
			time.Sleep(time.Millisecond * 100)
			assert.NoError(t, lock.Release(ctx))
		}()

		//this goroutine should wait until first lock is release
		go func() {
			waitingCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
			defer cancel()
			anotherInstanceMutex := mutexFabric()
			_, err := anotherInstanceMutex.Lock(waitingCtx, id)
			assert.NoError(t, err)
			select {
			case <-ctx.Done():
				return
			case releaseCh <- struct{}{}:
				return
			}
		}()

		select {
		case <-ctx.Done():
			t.FailNow()
		case <-releaseCh:
		}
	})

	t.Run("failed to acquire a lock", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		releaseCh := make(chan struct{})

		id := "aaa"
		lock, err := sqlMutex.Lock(ctx, id)
		assert.NoError(t, err)

		//this goroutine should wait until first lock is release
		go func() {
			waitingCtx, cancel := context.WithTimeout(ctx, time.Millisecond*200)
			defer cancel()

			_, err := sqlMutex.Lock(waitingCtx, id)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "context canceled while waiting for connection lock")
			select {
			case <-ctx.Done():
				return
			case releaseCh <- struct{}{}:
				return
			}
		}()

		select {
		case <-time.After(time.Millisecond * 500):
			t.FailNow()
		case <-releaseCh:
		}

		assert.NoError(t, lock.Release(ctx))
	})

	//t.Run("release not existing lock", func(t *testing.T) {
	//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	//	defer cancel()
	//
	//	id := "bbb"
	//
	//	lock, err := sqlMutex.Lock(ctx, id)
	//	require.NoError(t, err)
	//	assert.NoError(t, lock.Release(ctx))
	//
	//	assert.Error(t, lock.Release(ctx))
	//	assert.Contains(t, err.Error(), "connection which acquiring lock is not found in runtime map. Was Release() called after processing a message?")
	//})

	t.Run("acquire and release a lot of mutex", func(t *testing.T) {
		rand.Seed(time.Now().UnixNano())
		locksCount := 100

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		locks := make(chan mutex.Lock)
		doneCh := make(chan struct{})

		go func() {
			for l := range locks {
				go func(lock mutex.Lock) {
					time.Sleep(time.Duration(rand.Intn(500-100)+100) * time.Millisecond)
					releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
					defer cancel()
					assert.NoError(t, lock.Release(releaseCtx))
				}(l)
			}

			doneCh <- struct{}{}
		}()

		for i := 0; i < locksCount; i++ {
			lock, err := sqlMutex.Lock(ctx, fmt.Sprintf("said-%d", i))
			require.NoError(t, err)
			locks <- lock
		}

		close(locks)

		select {
		case <-ctx.Done():
			t.FailNow()
		case <-doneCh:
			return
		}
	})
}
