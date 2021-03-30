package saga

import (
	"context"
	"fmt"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/go-foreman/foreman/saga"
	intSuite "github.com/go-foreman/foreman/testing/integration/saga/suite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"strings"
	"testing"
	"time"
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
	schemeRegistry.RegisterTypes(&FilterSaga{})
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
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, mysqlStore.Create(ctx, sagaInstance))
		fetchedSagaInstance, err := mysqlStore.GetById(ctx, sagaInstance.ID())
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstance)
		assert.EqualValues(t, sagaInstance, fetchedSagaInstance)
	})

	t.Run("delete saga instance", func(t *testing.T) {
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, mysqlStore.Create(ctx, sagaInstance))
		assert.NoError(t, mysqlStore.Delete(ctx, sagaInstance.ID()))
		err = mysqlStore.Delete(ctx, "xxx")
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("no saga instance %s found", "xxx"))
	})

	//this test copies "Create saga instance test" because we don't use fixtures for testing right now and need a way to put records into db
	t.Run("find saga instance by id", func(t *testing.T) {
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, mysqlStore.Create(ctx, sagaInstance))

		foundSagaInstance, err := mysqlStore.GetById(ctx, sagaInstance.ID())
		assert.NoError(t, err)
		require.NotNil(t, foundSagaInstance)
		assert.EqualValues(t, sagaInstance, foundSagaInstance)

		unregisteredSaga := &UnregisteredSaga{WorkflowSaga: *workflowSaga}
		unregisteredSagaInstance := saga.NewSagaInstance(uuid.New().String(), "", unregisteredSaga)
		require.NoError(t, mysqlStore.Create(ctx, unregisteredSagaInstance))
		fetchedUnregisteredSaga, err := mysqlStore.GetById(ctx, unregisteredSagaInstance.ID())
		assert.Nil(t, fetchedUnregisteredSaga)
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("loading type %s for saga %s", scheme.GetStructTypeKey(UnregisteredSaga{}), unregisteredSagaInstance.ID())))

		//need to delete this unregistered saga because we won't be able to fetch it by id or using filters in the next tests
		err = mysqlStore.Delete(ctx, unregisteredSagaInstance.ID())
		require.NoError(t, err)
	})

	t.Run("find saga instance by filter", func(t *testing.T) {
		anotherSaga := &FilterSaga{WorkflowSaga{
			Field: "field",
			Value: "value",
		}}
		anotherSagaID := uuid.New().String()
		anotherSagaInstance := saga.NewSagaInstance(anotherSagaID, "", anotherSaga)
		require.NoError(t, mysqlStore.Create(ctx, anotherSagaInstance))
		require.NotNil(t, anotherSagaInstance)

		fetchedSagaInstances, err := mysqlStore.GetByFilter(ctx, saga.WithSagaId(anotherSagaID))
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstances)
		assert.Len(t, fetchedSagaInstances, 1)
		assert.EqualValues(t, anotherSagaInstance, fetchedSagaInstances[0])

		fetchedSagaInstances, err = mysqlStore.GetByFilter(ctx, saga.WithSagaType(scheme.GetStructTypeKey(FilterSaga{})))
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstances)
		assert.Len(t, fetchedSagaInstances, 1)
		assert.EqualValues(t, anotherSagaInstance, fetchedSagaInstances[0])

		fetchedSagaInstances, err = mysqlStore.GetByFilter(ctx, saga.WithStatus("created"))
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstances)
		assert.Len(t, fetchedSagaInstances, 3)

		noSagas, err := mysqlStore.GetByFilter(ctx, saga.WithSagaType("xxxx"))
		assert.NoError(t, err)
		require.NotNil(t, noSagas)
		assert.Len(t, noSagas, 0)

		noSagas, err = mysqlStore.GetByFilter(ctx, saga.WithSagaId("xxxx"))
		assert.NoError(t, err)
		require.NotNil(t, noSagas)
		assert.Len(t, noSagas, 0)

		noSagas, err = mysqlStore.GetByFilter(ctx, saga.WithStatus("completed"))
		assert.NoError(t, err)
		require.NotNil(t, noSagas)
		assert.Len(t, noSagas, 0)
	})

	t.Run("update saga instance", func(t *testing.T) {
		workflowSaga := &WorkflowSaga{Field: "field", Value: "value"}
		sagaInstance := saga.NewSagaInstance(uuid.New().String(), "", workflowSaga)
		require.NoError(t, mysqlStore.Create(ctx, sagaInstance))
		fetchedSagaInstance, err := mysqlStore.GetById(ctx, sagaInstance.ID())
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstance)
		assert.EqualValues(t, sagaInstance, fetchedSagaInstance)
		assert.Len(t, fetchedSagaInstance.HistoryEvents(), 0)

		someEv := &SomeEvent{Field: "field"}
		historyEvent := saga.HistoryEvent{
			Metadata:     message.Metadata{
				ID: uuid.New().String(),
				Type: message.EventType,
				Name: scheme.GetStructTypeKey(&SomeEvent{}),
			},
			CreatedAt:    time.Now().Round(time.Second).UTC(),
			Payload:      someEv,
			OriginSource: "originSource",
			SagaStatus:   "created",
			Description:  "something happened",
		}
		fetchedSagaInstance.AttachEvent(historyEvent)

		err = mysqlStore.Update(ctx, fetchedSagaInstance)
		require.NoError(t, err)
		fetchedSagaInstance, err = mysqlStore.GetById(ctx, sagaInstance.ID())
		//we get this error because SomeEvent is not registered in schema, let's register it
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), fmt.Sprintf("loading type %s for event %s", scheme.GetStructTypeKey(&SomeEvent{}), historyEvent.ID)) )
		schemeRegistry.RegisterTypes(&SomeEvent{})

		fetchedSagaInstance, err = mysqlStore.GetById(ctx, sagaInstance.ID())
		assert.NoError(t, err)
		require.NotNil(t, fetchedSagaInstance)
		require.Len(t, fetchedSagaInstance.HistoryEvents(), 1)
		assert.EqualValues(t, historyEvent, fetchedSagaInstance.HistoryEvents()[0])
	})
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

type FilterSaga struct {
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

func (a FilterSaga) EventHandlers() map[string]saga.Executor {
	panic("implement me")
}

type UnregisteredSaga struct {
	WorkflowSaga
}

type SomeEvent struct {
	Field string `json:"field"`
}
