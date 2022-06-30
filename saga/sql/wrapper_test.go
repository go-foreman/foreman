package sql

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSqlWrapper(t *testing.T) {
	t.Run("acquire lock and query conn, query does not release lock", func(t *testing.T) {
		wrapper, mock := createWrapper(t)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		sagaID := "123"
		lockConn, err := wrapper.Conn(ctx, sagaID, true)
		assert.NoError(t, err)
		mock.ExpectPing()

		queryConn, err := wrapper.Conn(ctx, sagaID, false)
		assert.NoError(t, err)
		assert.Same(t, lockConn, queryConn)
		assert.NoError(t, queryConn.Close(false))

		assert.Equal(t, lockConn.clients, uint32(1), "still one client is present")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to wait for the second lock, ctx cancelled", func(t *testing.T) {
		wrapper, mock := createWrapper(t)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		sagaID := "123"
		firstConn, err := wrapper.Conn(ctx, sagaID, true)
		assert.NoError(t, err)
		assert.NotNil(t, firstConn)

		secondCtx, secondCancel := context.WithTimeout(context.Background(), time.Microsecond*200)
		defer secondCancel()

		_, err = wrapper.Conn(secondCtx, sagaID, true)
		assert.Error(t, err)
		assert.EqualError(t, err, "acquiring connection: context canceled while waiting for connection lock")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("acquire, close, acquire again, not the same connection", func(t *testing.T) {
		wrapper, mock := createWrapper(t)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		sagaID := "123"
		firstConn, err := wrapper.Conn(ctx, sagaID, true)
		assert.NoError(t, err)
		assert.NotNil(t, firstConn)

		assert.NoError(t, firstConn.Close(true))

		secondConn, err := wrapper.Conn(ctx, sagaID, true)
		assert.NoError(t, err)
		assert.NotNil(t, firstConn)

		assert.NotSame(t, firstConn, secondConn)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wait for lock to be released and then acquire the same connection", func(t *testing.T) {
		wrapper, mock := createWrapper(t)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		sagaID := "123"
		firstConn, err := wrapper.Conn(ctx, sagaID, true)
		assert.NoError(t, err)
		assert.NotNil(t, firstConn)

		go func() {
			time.Sleep(time.Microsecond * 100)
			assert.NoError(t, firstConn.Close(true))
		}()

		secondCtx, secondCancel := context.WithTimeout(context.Background(), time.Microsecond*400)
		defer secondCancel()

		mock.ExpectPing()

		secondConn, err := wrapper.Conn(secondCtx, sagaID, true)
		assert.NoError(t, err)
		assert.NotNil(t, firstConn)

		assert.Same(t, firstConn, secondConn)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("connection is not alive", func(t *testing.T) {
		wrapper, mock := createWrapper(t)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		sagaID := "123"
		lockConn, err := wrapper.Conn(ctx, sagaID, true)
		assert.NoError(t, err)
		mock.ExpectPing().WillReturnError(errors.New("conn is dead"))

		queryConn, err := wrapper.Conn(ctx, sagaID, false)
		assert.NoError(t, err)
		assert.NotSame(t, lockConn, queryConn)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("unlock connection that wasn't locked", func(t *testing.T) {
		wrapper, _ := createWrapper(t)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		sagaID := "123"
		lockConn, err := wrapper.Conn(ctx, sagaID, false)
		assert.NoError(t, err)
		err = lockConn.Close(true)
		assert.Error(t, err)
		assert.EqualError(t, err, "called conn.Close(true) on connection that wasn't locked")
	})
}

func createWrapper(t *testing.T) (*DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	wrapper := NewDB(db)

	return wrapper, mock
}
