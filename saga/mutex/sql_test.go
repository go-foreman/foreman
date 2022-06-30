package mutex

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-foreman/foreman/saga"
	"github.com/go-foreman/foreman/saga/sql"
	"github.com/go-foreman/foreman/testing/log"
	"github.com/stretchr/testify/require"
)

func TestMysqlMutex(t *testing.T) {
	t.Run("successfully lock saga and unlock", func(t *testing.T) {
		m, mock, _ := createMutex(t, saga.MYSQLDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"

		//lock
		mock.
			ExpectQuery("SELECT GET_LOCK(?, -1);").
			WithArgs(sagaId).
			WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))

		lock, err := m.Lock(ctx, sagaId)
		assert.NoError(t, err)

		mock.
			ExpectQuery("SELECT RELEASE_LOCK(?);").
			WithArgs(sagaId).
			WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))

		assert.NoError(t, lock.Release(ctx))

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("lock query return an error", func(t *testing.T) {
		m, mock, _ := createMutex(t, saga.MYSQLDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"

		wantRows := sqlmock.NewRows([]string{"x"}).
			AddRow("-333")

		mock.
			ExpectQuery("SELECT GET_LOCK(?, -1);").
			WithArgs(sagaId).
			WillReturnRows(wantRows)

		_, err := m.Lock(ctx, sagaId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "got error status -333 when acquiring lock for saga 123")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query return an error", func(t *testing.T) {
		m, mock, _ := createMutex(t, saga.MYSQLDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"

		mock.
			ExpectQuery("SELECT GET_LOCK(?, -1);").
			WithArgs(sagaId).
			WillReturnError(errors.New("some error"))

		_, err := m.Lock(ctx, sagaId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "some error")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to release a lock, got an error", func(t *testing.T) {
		m, mock, _ := createMutex(t, saga.MYSQLDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"

		//lock
		mock.
			ExpectQuery("SELECT GET_LOCK(?, -1);").
			WithArgs(sagaId).
			WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))

		lock, err := m.Lock(ctx, sagaId)
		assert.NoError(t, err)

		//the first failed attempt releasing the lock
		mock.
			ExpectQuery("SELECT RELEASE_LOCK(?);").
			WithArgs(sagaId).
			WillReturnError(errors.New("release error"))

		err = lock.Release(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "release error")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to release a lock, got wrong status", func(t *testing.T) {
		m, mock, _ := createMutex(t, saga.MYSQLDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"

		//lock
		mock.
			ExpectQuery("SELECT GET_LOCK(?, -1);").
			WithArgs(sagaId).
			WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("1"))

		lock, err := m.Lock(ctx, sagaId)
		assert.NoError(t, err)

		//the second failed attempt releasing the lock
		mock.
			ExpectQuery("SELECT RELEASE_LOCK(?);").
			WithArgs(sagaId).
			WillReturnRows(sqlmock.NewRows([]string{"x"}).AddRow("-777"))

		err = lock.Release(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lock was not established by this thread for saga 123")

		// second release will have this connection closed
		err = lock.Release(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sql: connection is already closed")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPGMutex(t *testing.T) {
	t.Run("successfully lock saga and unlock", func(t *testing.T) {
		m, mock, testLogger := createMutex(t, saga.PGDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"

		mock.ExpectPing().WillReturnError(errors.New("ping returned an error"))
		mock.ExpectPing()

		//lock
		mock.
			ExpectExec("SELECT pg_advisory_lock(hashtext($1));").
			WithArgs(sagaId).
			WillReturnResult(sqlmock.NewResult(1, 1))

		lock, err := m.Lock(ctx, sagaId)
		assert.NoError(t, err)
		testLogger.AssertContainsSubstr(t, "error verifying that obtained mutex connection is alive")

		mock.
			ExpectExec("SELECT pg_advisory_unlock(hashtext($1));").
			WithArgs(sagaId).
			WillReturnResult(sqlmock.NewResult(1, 1))

		assert.NoError(t, lock.Release(ctx))

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("lock exec returns an error", func(t *testing.T) {
		m, mock, _ := createMutex(t, saga.PGDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"
		mock.ExpectPing()

		//lock
		mock.
			ExpectExec("SELECT pg_advisory_lock(hashtext($1));").
			WithArgs(sagaId).
			WillReturnError(errors.New("exec returned an error"))

		_, err := m.Lock(ctx, sagaId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exec returned an error")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("release exec returns an error", func(t *testing.T) {
		m, mock, _ := createMutex(t, saga.PGDriver)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		sagaId := "123"

		mock.ExpectPing()

		//lock
		mock.
			ExpectExec("SELECT pg_advisory_lock(hashtext($1));").
			WithArgs(sagaId).
			WillReturnResult(sqlmock.NewResult(1, 1))

		lock, err := m.Lock(ctx, sagaId)
		assert.NoError(t, err)

		mock.
			ExpectExec("SELECT pg_advisory_unlock(hashtext($1));").
			WithArgs(sagaId).
			WillReturnError(errors.New("release returns an error"))

		err = lock.Release(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "release returns an error")

		assert.NoError(t, mock.ExpectationsWereMet())
	})

}

func createMutex(t *testing.T, provider saga.SQLDriver) (Mutex, sqlmock.Sqlmock, *log.TestLogger) {
	db, mock, err := sqlmock.New(
		sqlmock.MonitorPingsOption(true),
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
	)
	require.NoError(t, err)
	wrapper := sql.NewDB(db)

	testLogger := log.NewNilLogger()

	return NewSqlMutex(wrapper, provider, testLogger), mock, testLogger
}
