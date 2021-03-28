package saga

import (
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"testing"
)

type mysqlStoreTest struct {
	intSuite.MysqlSuite
}

func TestMysqlStore(t *testing.T) {
	//MysqlSuite.Rn()
}


func TestMysqlSuite(t *testing.T) {
	mysqlStoreTest := &mysqlStoreTest{}
	suite.Run(t, mysqlStoreTest)
}

func (m *mysqlStoreTest) TestInit() {
	t := m.T()
	dbConnection := m.Connection()
	schemeRegistry := scheme.NewKnownTypesRegistry()
	mysqlStore, err := saga.NewMysqlSagaStore(dbConnection, schemeRegistry)
	require.NoError(t, err)
	require.NotNil(t, mysqlStore)

	t.Run("initialized store tables", func(t *testing.T) {
		res, err := dbConnection.Query("SELECT * from saga")
		require.NoError(t, err)
		require.NotNil(t, res)
		res, err = dbConnection.Query("SELECT * from saga_history")
		require.NoError(t, err)
		require.NotNil(t, res)
	})
}