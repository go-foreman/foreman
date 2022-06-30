package saga

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	sagaSql "github.com/go-foreman/foreman/saga/sql"

	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	testGroup scheme.Group = "testgroup"
)

type mysqlStoreTest struct {
	intSuite.MysqlSuite
}

func TestMysqlSuite(t *testing.T) {
	mysqlStoreTest := &mysqlStoreTest{}
	suite.Run(t, mysqlStoreTest)
}

func (m *mysqlStoreTest) TestMysqlStore() {
	t := m.T()

	schemeRegistry := scheme.NewKnownTypesRegistry()
	schemeRegistry.AddKnownTypes(testGroup, &WorkflowSaga{})
	schemeRegistry.AddKnownTypes(testGroup, &FilterSaga{})
	store, err := saga.NewSQLSagaStore(sagaSql.NewDB(m.Connection()), saga.MYSQLDriver, message.NewJsonMarshaller(schemeRegistry))

	require.NoError(t, err)
	require.NotNil(t, store)

	testSQLStoreUseCases(t, store, schemeRegistry, m.Connection())
}

func testSQLStoreUseCases(t *testing.T, store saga.Store, schemeRegistry scheme.KnownTypesRegistry, dbConnection *sql.DB) {
	t.Run("initialized store tables", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		res, err := dbConnection.QueryContext(ctx, "SELECT * from saga")
		require.NoError(t, err)
		require.NotNil(t, res)
		require.NoError(t, res.Close()) //nolint:sqlclosecheck
		res, err = dbConnection.QueryContext(ctx, "SELECT * from saga_history")
		require.NoError(t, err)
		require.NoError(t, res.Close()) //nolint:sqlclosecheck
	})

	t.Run("create saga instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, sagaInstance.Start(nil)) //started_at, updated_at are populated
		require.NoError(t, store.Create(ctx, sagaInstance))
		fetchedSagaInstance, err := store.GetById(ctx, sagaInstance.UID())
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstance)
		assert.EqualValues(t, sagaInstance, fetchedSagaInstance)
		require.NoError(t, store.Delete(ctx, sagaInstance.UID()))
	})

	t.Run("delete saga instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, store.Create(ctx, sagaInstance))
		assert.NoError(t, store.Delete(ctx, sagaInstance.UID()))
		err := store.Delete(ctx, "xxx")
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("no saga instance %s found", "xxx"))
	})

	//this test copies "Create saga instance test" because we don't use fixtures for testing right now and need a way to put records into db
	t.Run("find saga instance by id", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, store.Create(ctx, sagaInstance))

		foundSagaInstance, err := store.GetById(ctx, sagaInstance.UID())
		assert.NoError(t, err)
		require.NotNil(t, foundSagaInstance)
		assert.EqualValues(t, sagaInstance, foundSagaInstance)

		require.NoError(t, store.Delete(ctx, sagaInstance.UID()))

		unregisteredSaga := &UnregisteredSaga{WorkflowSaga: WorkflowSaga{Field: "x", Value: "y"}}
		unregisteredSagaInstance := saga.NewSagaInstance(uuid.New().String(), "", unregisteredSaga)
		err = store.Create(ctx, unregisteredSagaInstance)
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "no kind is registered in schema for the type UnregisteredSaga"))
		fetchedUnregisteredSaga, err := store.GetById(ctx, unregisteredSagaInstance.UID())
		assert.Nil(t, fetchedUnregisteredSaga)
		assert.NoError(t, err)
	})

	t.Run("update saga instance", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, store.Create(ctx, sagaInstance))
		fetchedSagaInstance, err := store.GetById(ctx, sagaInstance.UID())
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstance)
		assert.EqualValues(t, sagaInstance, fetchedSagaInstance)
		assert.Len(t, fetchedSagaInstance.HistoryEvents(), 0)

		someEv := &SomeEvent{Field: "field"}

		sagaInstance = fetchedSagaInstance
		fetchedSagaInstance.AddHistoryEvent(someEv, &saga.AddHistoryEvent{Origin: "origin", TraceUID: uuid.New().String()})

		err = store.Update(ctx, fetchedSagaInstance)
		require.Error(t, err)
		//we get this error because SomeEvent is not registered in schema, let's register it
		require.True(t, strings.Contains(err.Error(), "no kind is registered in schema for the type SomeEvent"))

		schemeRegistry.AddKnownTypes(testGroup, &SomeEvent{})

		require.NoError(t, store.Update(ctx, fetchedSagaInstance))

		fetchedSagaInstance, err = store.GetById(ctx, sagaInstance.UID())
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstance)
		require.Len(t, sagaInstance.HistoryEvents(), 1)
		require.Len(t, fetchedSagaInstance.HistoryEvents(), 1)
		assert.EqualValues(t, sagaInstance.HistoryEvents()[0], fetchedSagaInstance.HistoryEvents()[0])

		fetchedSagaInstance.Fail(&SomeEvent{
			ObjectMeta: message.ObjectMeta{
				TypeMeta: scheme.TypeMeta{
					Kind:  "SomeEvent",
					Group: testGroup.String(),
				},
			},
			Field: "failed",
		})

		require.NoError(t, store.Update(ctx, fetchedSagaInstance))
		failedSagaInstance, err := store.GetById(ctx, sagaInstance.UID())
		assert.NoError(t, err)
		require.NotNil(t, failedSagaInstance)
		assert.EqualValues(t, fetchedSagaInstance, failedSagaInstance)

		require.NoError(t, store.Delete(ctx, sagaInstance.UID()))
	})

	t.Run("find saga instance by filter", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		anotherSaga := &FilterSaga{WorkFlow: WorkflowSaga{
			Field: "field",
			Value: "value",
		}}
		anotherSagaInstance := saga.NewSagaInstance(uuid.New().String(), "xxx", anotherSaga)
		require.NoError(t, store.Create(ctx, anotherSagaInstance))
		require.NotNil(t, anotherSagaInstance)
		assert.EqualValues(t, anotherSaga, anotherSagaInstance.Saga())

		fetchedSagaInstances, err := store.GetByFilter(ctx, saga.WithSagaId(anotherSagaInstance.UID()))
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstances)
		assert.Len(t, fetchedSagaInstances, 1)
		assert.EqualValues(t, anotherSagaInstance, fetchedSagaInstances[0])

		gk, err := schemeRegistry.ObjectKind(anotherSaga)
		require.NoError(t, err)

		fetchedSagaInstances, err = store.GetByFilter(ctx, saga.WithSagaName(gk.String()))
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstances)
		assert.Len(t, fetchedSagaInstances, 1)
		assert.EqualValues(t, anotherSagaInstance, fetchedSagaInstances[0])

		fetchedSagaInstances, err = store.GetByFilter(ctx, saga.WithStatus("created"))
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstances)
		assert.Len(t, fetchedSagaInstances, 1)

		noSagas, err := store.GetByFilter(ctx, saga.WithSagaName("xxx"))
		assert.NoError(t, err)
		require.NotNil(t, noSagas)
		assert.Len(t, noSagas, 0)

		noSagas, err = store.GetByFilter(ctx, saga.WithSagaId("xxxx"))
		assert.NoError(t, err)
		require.NotNil(t, noSagas)
		assert.Len(t, noSagas, 0)

		noSagas, err = store.GetByFilter(ctx, saga.WithStatus("completed"))
		assert.NoError(t, err)
		require.NotNil(t, noSagas)
		assert.Len(t, noSagas, 0)
	})
}

type WorkflowSaga struct {
	saga.BaseSaga
	Field string `json:"field"`
	Value string `json:"value"`
}

func (w WorkflowSaga) Init() {
	panic("implement me")
}

func (w WorkflowSaga) Start(execCtx saga.SagaContext) error {
	return nil
}

func (w WorkflowSaga) Compensate(execCtx saga.SagaContext) error {
	return nil
}

func (w WorkflowSaga) Recover(execCtx saga.SagaContext) error {
	return nil
}

type FilterSaga struct {
	saga.BaseSaga
	WorkFlow WorkflowSaga `json:"work_flow"`
}

func (a FilterSaga) Init() {
	panic("implement me")
}

func (a FilterSaga) Start(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (a FilterSaga) Compensate(execCtx saga.SagaContext) error {
	panic("implement me")
}

func (a FilterSaga) Recover(execCtx saga.SagaContext) error {
	panic("implement me")
}

type UnregisteredSaga struct {
	WorkflowSaga
}

type SomeEvent struct {
	message.ObjectMeta
	Field string `json:"field"`
}
