package wraper

import (
	"context"
	"testing"
	"time"

	sagaSql "github.com/go-foreman/foreman/saga/sql"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type sqlWrapperTest struct {
	intSuite.MysqlSuite
}

func TestMysqlSuite(t *testing.T) {
	sqlWrapperTest := &sqlWrapperTest{}
	suite.Run(t, sqlWrapperTest)
}

func (m *sqlWrapperTest) TestSqlWrapperConn() {
	t := m.T()

	dbWrapper := sagaSql.NewDB(m.Connection())

	t.Run("while mutex is held it's possible to use the very same connection for queries or execs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		conn, err := dbWrapper.Conn(ctx, "xxx", true)
		require.NoError(t, err)
		require.NotNil(t, conn)

		shouldBeTheSameConn, err := dbWrapper.Conn(ctx, "xxx", false)
		require.NoError(t, err)
		require.NotNil(t, shouldBeTheSameConn)
		shouldBeTheSameConn.Close(false)

		shouldBeTheSameConn2, err := dbWrapper.Conn(ctx, "xxx", false)
		require.NoError(t, err)
		require.NotNil(t, shouldBeTheSameConn2)
		shouldBeTheSameConn2.Close(false)

		assert.Same(t, conn, shouldBeTheSameConn)
		assert.Same(t, conn, shouldBeTheSameConn2)

		conn.Close(true) // at this step the connection must have been closed and deleted from the map

		againConn, err := dbWrapper.Conn(ctx, "xxx", true)
		require.NoError(t, err)
		require.NotNil(t, againConn)
		assert.NotSame(t, conn, againConn)

		againConn.Close(true)

	})

	t.Run("wait until mutex is released", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		conn, err := dbWrapper.Conn(ctx, "yyy", true)
		require.NoError(t, err)
		require.NotNil(t, conn)
		waitingCtx, waitingCancel := context.WithTimeout(ctx, time.Millisecond*300)
		defer waitingCancel()

		go func() {
			time.Sleep(time.Millisecond * 100)
			t.Log(conn.Close(true))
		}()

		shouldBeTheSameConn, err := dbWrapper.Conn(waitingCtx, "yyy", true)
		require.NoError(t, err)
		require.NotNil(t, shouldBeTheSameConn)
		defer func() {
			t.Log(shouldBeTheSameConn.Close(true))
		}()

		assert.Same(t, conn, shouldBeTheSameConn)
	})

	t.Run("context canceled when was waiting to obtain a mutex", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		conn, err := dbWrapper.Conn(ctx, "zzz", true)

		require.NoError(t, err)
		require.NotNil(t, conn)
		waitingCtx, waitingCancel := context.WithTimeout(ctx, time.Millisecond*100)
		defer waitingCancel()

		go func() {
			time.Sleep(time.Millisecond * 200)
			t.Log(conn.Close(true))
		}()

		shouldBeTheSameConn, err := dbWrapper.Conn(waitingCtx, "zzz", true)
		require.Error(t, err)
		require.Nil(t, shouldBeTheSameConn)
		assert.EqualError(t, err, "acquiring connection: context canceled while waiting for connection lock")
	})
}
