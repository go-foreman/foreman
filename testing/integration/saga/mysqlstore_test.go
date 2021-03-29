package saga

import (
	"context"
	"fmt"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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

func (m *mysqlStoreTest) TestMysqlStore() {
	t := m.T()
	ctx := context.Background()
	dbConnection := m.Connection()
	schemeRegistry := scheme.NewKnownTypesRegistry()
	schemeRegistry.RegisterTypes(&WorkflowSaga{})
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

	t.Run("create saga instance", func(t *testing.T) {
		t.Parallel()
		sagaID := uuid.New().String()
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(sagaID, "", workflowSaga)
		require.NoError(t, mysqlStore.Create(ctx, sagaInstance))
		fetchedSagaInstance, err := mysqlStore.GetById(ctx, sagaID)
		assert.NoError(t, err)
		assert.NotNil(t, fetchedSagaInstance)
		assert.EqualValues(t, sagaInstance, fetchedSagaInstance)
		require.NoError(t, mysqlStore.Delete(ctx, sagaID))
	})

	t.Run("delete saga instance", func(t *testing.T) {
		sagaID := uuid.New().String()
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(sagaID, "", workflowSaga)
		require.NoError(t, mysqlStore.Create(ctx, sagaInstance))
		assert.NoError(t, mysqlStore.Delete(ctx, sagaID))
		err = mysqlStore.Delete(ctx, "xxx")
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("no saga instance %s found", "xxx"))
	})

	//this test copies "Create saga instance test" because we don't use fixtures for testing right now and need a way to put records into db
	t.Run("find saga instance by id", func(t *testing.T) {
		sagaID := uuid.New().String()
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(sagaID, "", workflowSaga)
		require.NoError(t, mysqlStore.Create(ctx, sagaInstance))

		foundSagaInstance, err := mysqlStore.GetById(ctx, sagaID)
		assert.NoError(t, err)
		assert.NotNil(t, foundSagaInstance)
		assert.EqualValues(t, sagaInstance, foundSagaInstance)
	})

	//t.Run("find saga instance by filter", func(t *testing.T) {
	//	anotherSaga := &AnotherSaga{WorkflowSaga{
	//		Field: "field",
	//		Value: "value",
	//	}}
	//	anotherSagaID := uuid.New().String()
	//	anotherSagaInstance := saga.NewSagaInstance(anotherSagaID, "", anotherSaga)
	//	require.NoError(t, mysqlStore.Create(ctx, anotherSagaInstance))
	//	require.NotNil(t, anotherSagaInstance)
	//
	//	fetchedSagaInstances, err := mysqlStore.GetByFilter(ctx, saga.WithSagaId(anotherSagaID))
	//	assert.NoError(t, err)
	//	assert.NotNil(t, fetchedSagaInstances)
	//	assert.Len(t, fetchedSagaInstances, 1)
	//	assert.EqualValues(t, anotherSagaInstance, fetchedSagaInstances[0])
	//})
}

type WorkflowSaga struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

func (w WorkflowSaga) Init() {
	panic("implement me")
}

func (w WorkflowSaga) Start(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (w WorkflowSaga) Compensate(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (w WorkflowSaga) Recover(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (w WorkflowSaga) EventHandlers() map[string]saga.Executor {
	panic("implement me")
}

type AnotherSaga struct {
	WorkFlow WorkflowSaga `json:"work_flow"`
}

func (a AnotherSaga) Init() {
	panic("implement me")
}

func (a AnotherSaga) Start(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (a AnotherSaga) Compensate(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (a AnotherSaga) Recover(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (a AnotherSaga) EventHandlers() map[string]saga.Executor {
	panic("implement me")
}
