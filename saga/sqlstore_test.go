package saga

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-foreman/foreman/runtime/scheme"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-foreman/foreman/pubsub/message"
	formanSql "github.com/go-foreman/foreman/saga/sql"
	mockMessage "github.com/go-foreman/foreman/testing/mocks/pubsub/message"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestSqlStore_InitTable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	msgMarshallerMock := mockMessage.NewMockMarshaller(ctrl)

	t.Run("error committing", func(t *testing.T) {
		db, mock, err := sqlmock.New(
			sqlmock.MonitorPingsOption(true),
			sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
		)
		require.NoError(t, err)
		wrapper := formanSql.NewDB(db)

		mock.ExpectBegin()
		mock.ExpectExec("create table if not exists saga ( uid varchar(255) not null primary key, parent_uid varchar(255) null, name varchar(255) null, payload text null, status varchar(255) null, started_at timestamp null, updated_at timestamp null, last_failed_ev text null );").
			WithArgs().
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("create table if not exists saga_history ( uid varchar(255) not null primary key, saga_uid varchar(255) not null, name varchar(255) null, status varchar(255) null, payload text null, origin varchar(255) null, created_at timestamp null, trace_uid varchar(255) null, constraint saga_history_saga_model_id_fk foreign key (saga_uid) references saga (uid) on update cascade on delete cascade );").
			WithArgs().
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit().WillReturnError(errors.New("error commit"))

		_, err = NewSQLSagaStore(wrapper, MYSQLDriver, msgMarshallerMock)
		require.Error(t, err)
		assert.EqualError(t, err, "initializing tables for SQLSagaStore, driver mysql: error commit")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error exec saga table", func(t *testing.T) {
		db, mock, err := sqlmock.New(
			sqlmock.MonitorPingsOption(true),
			sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
		)
		require.NoError(t, err)
		wrapper := formanSql.NewDB(db)

		mock.ExpectBegin()
		mock.ExpectExec("create table if not exists saga ( uid varchar(255) not null primary key, parent_uid varchar(255) null, name varchar(255) null, payload text null, status varchar(255) null, started_at timestamp null, updated_at timestamp null, last_failed_ev text null );").
			WithArgs().
			WillReturnError(errors.New("error exec1"))
		mock.ExpectRollback()

		_, err = NewSQLSagaStore(wrapper, MYSQLDriver, msgMarshallerMock)
		require.Error(t, err)
		assert.EqualError(t, err, "initializing tables for SQLSagaStore, driver mysql: error exec1")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error exec saga history table", func(t *testing.T) {
		db, mock, err := sqlmock.New(
			sqlmock.MonitorPingsOption(true),
			sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
		)
		require.NoError(t, err)
		wrapper := formanSql.NewDB(db)

		mock.ExpectBegin()
		mock.ExpectExec("create table if not exists saga ( uid varchar(255) not null primary key, parent_uid varchar(255) null, name varchar(255) null, payload text null, status varchar(255) null, started_at timestamp null, updated_at timestamp null, last_failed_ev text null );").
			WithArgs().
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("create table if not exists saga_history ( uid varchar(255) not null primary key, saga_uid varchar(255) not null, name varchar(255) null, status varchar(255) null, payload text null, origin varchar(255) null, created_at timestamp null, trace_uid varchar(255) null, constraint saga_history_saga_model_id_fk foreign key (saga_uid) references saga (uid) on update cascade on delete cascade );").
			WithArgs().
			WillReturnError(errors.New("error exec2"))
		mock.ExpectRollback()

		_, err = NewSQLSagaStore(wrapper, MYSQLDriver, msgMarshallerMock)
		require.Error(t, err)
		assert.EqualError(t, err, "initializing tables for SQLSagaStore, driver mysql: error exec2")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

}

func TestSqlStore_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	sagaID := "123"
	parentSagaID := "321"

	t.Run("mysql create saga", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, MYSQLDriver)
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, &SagaExample{Data: "data"})
		payload := []byte("payload")

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(payload, nil)

		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO saga (uid, parent_uid, name, payload, status, started_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?);").
			WithArgs(
				sagaInstance.UID(),
				sagaInstance.ParentID(),
				sagaInstance.Saga().GroupKind().String(),
				payload,
				sagaInstance.Status().String(),
				sagaInstance.StartedAt(),
				sagaInstance.UpdatedAt(),
			).WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		err := store.Create(ctx, sagaInstance)
		assert.NoError(t, err)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("pg create saga", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, PGDriver)
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, &SagaExample{Data: "data"})

		payload := []byte("payload")

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(payload, nil)

		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO saga (uid, parent_uid, name, payload, status, started_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7);").
			WithArgs(
				sagaInstance.UID(),
				sagaInstance.ParentID(),
				sagaInstance.Saga().GroupKind().String(),
				payload,
				sagaInstance.Status().String(),
				sagaInstance.StartedAt(),
				sagaInstance.UpdatedAt(),
			).WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		err := store.Create(ctx, sagaInstance)
		assert.NoError(t, err)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("error marshaling saga", func(t *testing.T) {
		store, _, marshallerMock := createStore(t, ctrl, PGDriver)
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, &SagaExample{Data: "data"})

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(nil, errors.New("error marshaling"))

		err := store.Create(ctx, sagaInstance)
		assert.Error(t, err)
		assert.EqualError(t, err, "error marshaling")
	})

	t.Run("error exec and rollback", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, MYSQLDriver)
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, &SagaExample{Data: "data"})

		payload := []byte("payload")

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(payload, nil)

		dbMock.ExpectBegin()
		dbMock.ExpectExec("INSERT INTO saga (uid, parent_uid, name, payload, status, started_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?);").
			WithArgs(
				sagaInstance.UID(),
				sagaInstance.ParentID(),
				sagaInstance.Saga().GroupKind().String(),
				payload,
				sagaInstance.Status().String(),
				sagaInstance.StartedAt(),
				sagaInstance.UpdatedAt(),
			).WillReturnError(errors.New("exec error"))
		dbMock.ExpectRollback()

		err := store.Create(ctx, sagaInstance)
		assert.Error(t, err)
		assert.EqualError(t, err, "inserting saga instance 123: exec error")
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("error beginning Tx", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, MYSQLDriver)
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, &SagaExample{Data: "data"})

		payload := []byte("payload")

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(payload, nil)

		dbMock.ExpectBegin().WillReturnError(errors.New("error Begin"))

		err := store.Create(ctx, sagaInstance)
		assert.Error(t, err)
		assert.EqualError(t, err, "beginning a transaction for saga 123: error Begin")
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestSqlStore_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	sagaID := "123"
	parentSagaID := "321"

	t.Run("error marshalling", func(t *testing.T) {
		store, _, marshallerMock := createStore(t, ctrl, PGDriver)
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, &SagaExample{Data: "data"})

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(nil, errors.New("error marshaling"))

		err := store.Update(ctx, sagaInstance)
		assert.Error(t, err)
		assert.EqualError(t, err, "marshaling saga instance 123 on update: error marshaling")

		sagaInstance.Fail(&ExampleEv{Data: "failed"})
		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return([]byte("payload"), nil)
		marshallerMock.
			EXPECT().
			Marshal(&ExampleEv{Data: "failed"}).
			Return(nil, errors.New("error marshaling failed ev"))

		err = store.Update(ctx, sagaInstance)
		assert.Error(t, err)
		assert.EqualError(t, err, "marshaling last failed event of saga instance 123 on update: error marshaling failed ev")
	})

	t.Run("mysql update saga", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, MYSQLDriver)
		sagaObj := &SagaExample{
			BaseSaga: BaseSaga{
				ObjectMeta: message.ObjectMeta{
					TypeMeta: scheme.TypeMeta{
						Kind:  "SagaExample",
						Group: "example",
					},
				},
				adjacencyMap: nil,
				scheme:       nil,
			},
			Data: "data",
		}
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, sagaObj)
		sagaInstance.AddHistoryEvent(&ExampleEv{Data: "h1"}, &AddHistoryEvent{
			TraceUID: "xxx",
			Origin:   "yyy",
		})
		sagaInstance.AddHistoryEvent(&ExampleEv{Data: "h2"}, &AddHistoryEvent{
			TraceUID: "qqq",
			Origin:   "ttt",
		})

		sagaInstance.Fail(&ExampleEv{Data: "failed"})

		payload := []byte("payload")

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(payload, nil)

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Status().FailedOnEvent()).
			Return(payload, nil)

		require.Len(t, sagaInstance.HistoryEvents(), 2)
		ev := sagaInstance.HistoryEvents()[0]

		marshallerMock.
			EXPECT().
			Marshal(ev.Payload).
			Return(payload, nil)

		dbMock.ExpectBegin()
		dbMock.ExpectExec("UPDATE saga SET parent_uid=?, name=?, payload=?, status=?, started_at=?, updated_at=?, last_failed_ev=? WHERE uid=?;").
			WithArgs(
				sagaInstance.ParentID(),
				"example.SagaExample",
				payload,
				sagaInstance.Status().String(),
				sagaInstance.StartedAt(),
				sagaInstance.UpdatedAt(),
				payload,
				sagaInstance.UID(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectQuery("SELECT uid FROM saga_history WHERE saga_uid=?;").
			WithArgs(sagaInstance.UID()).
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow(sagaInstance.HistoryEvents()[1].UID))

		dbMock.ExpectExec("INSERT INTO saga_history (uid, saga_uid, name, status, payload, origin, created_at, trace_uid) VALUES (?, ?, ?, ?, ?, ?, ?, ?);").
			WithArgs(
				ev.UID,
				sagaInstance.UID(),
				ev.Payload.GroupKind().String(),
				ev.SagaStatus,
				payload,
				ev.OriginSource,
				ev.CreatedAt,
				ev.TraceUID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock.ExpectCommit()

		assert.NoError(t, store.Update(ctx, sagaInstance))

		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("pg update saga", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, PGDriver)
		sagaObj := &SagaExample{
			BaseSaga: BaseSaga{
				ObjectMeta: message.ObjectMeta{
					TypeMeta: scheme.TypeMeta{
						Kind:  "SagaExample",
						Group: "example",
					},
				},
				adjacencyMap: nil,
				scheme:       nil,
			},
			Data: "data",
		}
		sagaInstance := NewSagaInstance(sagaID, parentSagaID, sagaObj)
		sagaInstance.AddHistoryEvent(&ExampleEv{Data: "h1"}, &AddHistoryEvent{
			TraceUID: "xxx",
			Origin:   "yyy",
		})
		sagaInstance.AddHistoryEvent(&ExampleEv{Data: "h2"}, &AddHistoryEvent{
			TraceUID: "qqq",
			Origin:   "ttt",
		})

		sagaInstance.Fail(&ExampleEv{Data: "failed"})

		payload := []byte("payload")

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Saga()).
			Return(payload, nil)

		marshallerMock.
			EXPECT().
			Marshal(sagaInstance.Status().FailedOnEvent()).
			Return(payload, nil)

		require.Len(t, sagaInstance.HistoryEvents(), 2)
		ev := sagaInstance.HistoryEvents()[0]

		marshallerMock.
			EXPECT().
			Marshal(ev.Payload).
			Return(payload, nil)

		dbMock.ExpectBegin()
		dbMock.ExpectExec("UPDATE saga SET parent_uid=$1, name=$2, payload=$3, status=$4, started_at=$5, updated_at=$6, last_failed_ev=$7 WHERE uid=$8;").
			WithArgs(
				sagaInstance.ParentID(),
				"example.SagaExample",
				payload,
				sagaInstance.Status().String(),
				sagaInstance.StartedAt(),
				sagaInstance.UpdatedAt(),
				payload,
				sagaInstance.UID(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectQuery("SELECT uid FROM saga_history WHERE saga_uid=$1;").
			WithArgs(sagaInstance.UID()).
			WillReturnRows(sqlmock.NewRows([]string{"uid"}).AddRow(sagaInstance.HistoryEvents()[1].UID))

		dbMock.ExpectExec("INSERT INTO saga_history (uid, saga_uid, name, status, payload, origin, created_at, trace_uid) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);").
			WithArgs(
				ev.UID,
				sagaInstance.UID(),
				ev.Payload.GroupKind().String(),
				ev.SagaStatus,
				payload,
				ev.OriginSource,
				ev.CreatedAt,
				ev.TraceUID,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		dbMock.ExpectCommit()

		assert.NoError(t, store.Update(ctx, sagaInstance))

		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestSqlStore_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	sagaID := "123"

	t.Run("mysql delete saga by id", func(t *testing.T) {
		store, dbMock, _ := createStore(t, ctrl, MYSQLDriver)

		dbMock.ExpectExec("DELETE FROM saga WHERE uid=?;").
			WithArgs(sagaID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.Delete(ctx, sagaID)
		assert.NoError(t, err)
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("pg delete saga by id no rows affected", func(t *testing.T) {
		store, dbMock, _ := createStore(t, ctrl, PGDriver)

		dbMock.ExpectExec("DELETE FROM saga WHERE uid=$1;").
			WithArgs(sagaID).
			WillReturnResult(sqlmock.NewResult(1, 0))

		err := store.Delete(ctx, sagaID)
		assert.Error(t, err)
		assert.EqualError(t, err, "no saga instance 123 found")
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("exec returns an error", func(t *testing.T) {
		store, dbMock, _ := createStore(t, ctrl, PGDriver)

		dbMock.ExpectExec("DELETE FROM saga WHERE uid=$1;").
			WithArgs(sagaID).
			WillReturnError(errors.New("exec error"))

		err := store.Delete(ctx, sagaID)
		assert.Error(t, err)
		assert.EqualError(t, err, "executing delete query for saga 123: exec error")
		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestSqlStore_GetById(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	sagaID := "123"

	t.Run("mysql get saga by id", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, MYSQLDriver)

		timeNow := time.Now()

		sagaData := &sagaSqlModel{
			ID: sql.NullString{
				String: "123",
				Valid:  true,
			},
			ParentID: sql.NullString{
				String: "222",
				Valid:  true,
			},
			Name: sql.NullString{
				String: "example.SagaExample",
				Valid:  true,
			},
			Payload: []byte("payload"),
			Status: sql.NullString{
				String: "failed",
				Valid:  true,
			},
			LastFailedMsg: []byte("payload"),
			StartedAt: sql.NullTime{
				Time:  timeNow,
				Valid: true,
			},
			UpdatedAt: sql.NullTime{
				Time:  timeNow,
				Valid: true,
			},
		}

		dbMock.ExpectQuery("SELECT s.uid, s.parent_uid, s.name, s.payload, s.status, s.last_failed_ev, s.started_at, s.updated_at FROM saga s WHERE uid=?;").
			WithArgs(sagaID).
			WillReturnRows(
				sqlmock.NewRows([]string{"uid", "parent_uid", "name", "payload", "status", "last_failed_ev", "started_at", "updated_at"}).
					AddRow(
						sagaData.ID.String,
						sagaData.ParentID.String,
						sagaData.Name.String,
						sagaData.Payload,
						sagaData.Status.String,
						sagaData.LastFailedMsg,
						sagaData.StartedAt.Time,
						sagaData.UpdatedAt.Time,
					),
			)

		evData := &historyEventSqlModel{
			ID: sql.NullString{
				String: "xxx",
				Valid:  true,
			},
			Name: sql.NullString{
				String: "example.DataExample",
				Valid:  true,
			},
			CreatedAt: sql.NullTime{
				Time:  timeNow,
				Valid: true,
			},
			Payload: []byte("payload"),
			OriginSource: sql.NullString{
				String: "origin",
				Valid:  true,
			},
			SagaStatus: sql.NullString{
				String: "created",
				Valid:  true,
			},
			TraceUID: sql.NullString{
				String: "gg",
				Valid:  true,
			},
		}

		dbMock.ExpectQuery("SELECT uid, name, status, payload, origin, created_at, trace_uid FROM saga_history WHERE saga_uid=? ORDER BY created_at;").
			WithArgs(sagaID).
			WillReturnRows(
				sqlmock.NewRows([]string{"uid", "name", "status", "payload", "origin", "created_at", "trace_uid"}).
					AddRow(evData.ID.String, evData.Name.String, evData.SagaStatus.String, evData.Payload, evData.OriginSource.String, evData.CreatedAt.Time, evData.TraceUID.String),
			)

		marshallerMock.
			EXPECT().
			Unmarshal(sagaData.LastFailedMsg).
			Return(&DataContract{Message: "data"}, nil)
		marshallerMock.
			EXPECT().
			Unmarshal(sagaData.Payload).
			Return(&SagaExample{Data: "data"}, nil)
		marshallerMock.
			EXPECT().
			Unmarshal(evData.Payload).
			Return(&DataContract{Message: "h1"}, nil)

		sagaInstance, err := store.GetById(ctx, sagaID)
		assert.NoError(t, err)
		assert.NotNil(t, sagaInstance)

		assert.Equal(t, sagaData.ID.String, sagaInstance.UID())
		assert.Equal(t, sagaData.ParentID.String, sagaInstance.ParentID())
		assert.Equal(t, sagaData.Status.String, sagaInstance.Status().String())
		assert.Equal(t, &SagaExample{Data: "data"}, sagaInstance.Saga())
		require.Len(t, sagaInstance.HistoryEvents(), 1)
		ev := sagaInstance.HistoryEvents()[0]

		assert.Equal(t, evData.ID.String, ev.UID)
		assert.Equal(t, evData.TraceUID.String, ev.TraceUID)
		assert.Equal(t, evData.SagaStatus.String, ev.SagaStatus)
		assert.Equal(t, evData.OriginSource.String, ev.OriginSource)
		assert.Equal(t, evData.CreatedAt.Time, ev.CreatedAt)
		assert.Equal(t, &DataContract{Message: "h1"}, ev.Payload)

		assert.NoError(t, dbMock.ExpectationsWereMet())
	})

	t.Run("PG: no saga found", func(t *testing.T) {
		store, dbMock, _ := createStore(t, ctrl, PGDriver)

		dbMock.ExpectQuery("SELECT s.uid, s.parent_uid, s.name, s.payload, s.status, s.last_failed_ev, s.started_at, s.updated_at FROM saga s WHERE uid=$1;").
			WithArgs(sagaID).
			WillReturnRows(
				sqlmock.NewRows([]string{"uid", "parent_uid", "name", "payload", "status", "last_failed_ev", "started_at", "updated_at"}),
			)

		sagaInstance, err := store.GetById(ctx, sagaID)
		assert.NoError(t, err)
		assert.Nil(t, sagaInstance)

		assert.NoError(t, dbMock.ExpectationsWereMet())
	})
}

func TestSqlStore_GetByFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("no filters specified", func(t *testing.T) {
		store, _, _ := createStore(t, ctrl, MYSQLDriver)

		_, err := store.GetByFilter(ctx)
		assert.Error(t, err)
		assert.EqualError(t, err, "no filters found, you have to specify at least one so result won't be whole store")
	})

	t.Run("empty values in filters", func(t *testing.T) {
		store, _, _ := createStore(t, ctrl, MYSQLDriver)

		_, err := store.GetByFilter(ctx, WithStatus(""), WithSagaName(""), WithSagaId(""))
		assert.Error(t, err)
		assert.EqualError(t, err, "all specified filters are empty, you have to specify at least one so result won't be whole store")
	})

	t.Run("get with all filters", func(t *testing.T) {
		store, dbMock, marshallerMock := createStore(t, ctrl, MYSQLDriver)

		timeNow := time.Now()

		sagaData := &sagaSqlModel{
			ID: sql.NullString{
				String: "sagaId",
				Valid:  true,
			},
			ParentID: sql.NullString{
				String: "parentSagaId",
				Valid:  true,
			},
			Name: sql.NullString{
				String: "example.SagaExample",
				Valid:  true,
			},
			Payload: []byte("payload"),
			Status: sql.NullString{
				String: "failed",
				Valid:  true,
			},
			LastFailedMsg: []byte("payload"),
			StartedAt: sql.NullTime{
				Time:  timeNow,
				Valid: true,
			},
			UpdatedAt: sql.NullTime{
				Time:  timeNow,
				Valid: true,
			},
		}

		evData1 := &historyEventSqlModel{
			ID: sql.NullString{
				String: "xxx",
				Valid:  true,
			},
			SagaUID: sql.NullString{
				String: "sagaId",
				Valid:  true,
			},
			Name: sql.NullString{
				String: "example.DataExample",
				Valid:  true,
			},
			CreatedAt: sql.NullTime{
				Time:  timeNow,
				Valid: true,
			},
			Payload: []byte("payload"),
			OriginSource: sql.NullString{
				String: "origin",
				Valid:  true,
			},
			SagaStatus: sql.NullString{
				String: "created",
				Valid:  true,
			},
			TraceUID: sql.NullString{
				String: "gg",
				Valid:  true,
			},
		}
		evData2 := evData1
		evData2.ID.String = "yyy"

		dbMock.ExpectQuery("SELECT COUNT(s.uid) cnt FROM saga s WHERE s.uid = ? AND s.status = ? AND s.name = ?;").
			WithArgs("sagaId", "created", "sagaName").
			WillReturnRows(
				sqlmock.NewRows([]string{"cnt"}).
					AddRow(1),
			)

		dbMock.ExpectQuery("SELECT s.uid, s.parent_uid, s.name, s.payload, s.status, s.last_failed_ev, s.started_at, s.updated_at FROM saga s  WHERE s.uid = ? AND s.status = ? AND s.name = ? ORDER BY started_at DESC;").
			WithArgs("sagaId", "created", "sagaName").
			WillReturnRows(
				sqlmock.NewRows([]string{
					"s.uid", "s.parent_uid", "s.name", "s.payload", "s.status", "s.last_failed_ev", "s.started_at", "s.updated_at",
				}).AddRow(
					sagaData.ID.String,
					sagaData.ParentID.String,
					sagaData.Name.String,
					sagaData.Payload,
					sagaData.Status.String,
					sagaData.LastFailedMsg,
					sagaData.StartedAt.Time,
					sagaData.UpdatedAt.Time,
				),
			)

		dbMock.ExpectQuery(`SELECT sh.uid, sh.saga_uid, sh.name, sh.status, sh.payload, sh.origin, sh.created_at, sh.trace_uid FROM saga_history sh WHERE sh.saga_uid IN (?);`).
			WithArgs("sagaId").
			WillReturnRows(
				sqlmock.NewRows([]string{
					"sh.uid", "sh.saga_uid,", "sh.name", "sh.status", "sh.payload", "sh.origin", "sh.created_at", "sh.trace_uid",
				}).AddRow(
					evData1.ID.String,
					evData1.SagaUID.String,
					evData1.Name.String,
					evData1.SagaStatus.String,
					evData1.Payload,
					evData1.OriginSource.String,
					evData1.CreatedAt.Time,
					evData1.TraceUID.String,
				).AddRow(
					evData2.ID.String,
					evData1.SagaUID.String,
					evData2.Name.String,
					evData2.SagaStatus.String,
					evData2.Payload,
					evData2.OriginSource.String,
					evData2.CreatedAt.Time,
					evData2.TraceUID.String,
				),
			)

		marshallerMock.
			EXPECT().
			Unmarshal(sagaData.LastFailedMsg).
			Return(&DataContract{Message: "failed-ev"}, nil)

		marshallerMock.
			EXPECT().
			Unmarshal(sagaData.Payload).
			Return(&SagaExample{Data: "saga"}, nil)
		marshallerMock.
			EXPECT().
			Unmarshal(evData1.Payload).
			Return(&DataContract{Message: "h1"}, nil)
		marshallerMock.
			EXPECT().
			Unmarshal(evData2.Payload).
			Return(&DataContract{Message: "h2"}, nil)
		sagas, err := store.GetByFilter(ctx, WithSagaId("sagaId"), WithStatus("created"), WithSagaName("sagaName"))
		assert.NoError(t, err)
		require.Equal(t, sagas.Total, 1)
		require.Len(t, sagas.Items, 1)
		assert.Equal(t, sagaData.ID.String, sagas.Items[0].UID())
		assert.Equal(t, sagaData.ParentID.String, sagas.Items[0].ParentID())
		assert.Equal(t, sagaData.Status.String, sagas.Items[0].Status().String())
		assert.Equal(t, &SagaExample{Data: "saga"}, sagas.Items[0].Saga())

		assert.Len(t, sagas.Items[0].HistoryEvents(), 2)
	})
}

func createStore(t *testing.T, ctrl *gomock.Controller, provider SQLDriver) (Store, sqlmock.Sqlmock, *mockMessage.MockMarshaller) {
	db, mock, err := sqlmock.New(
		sqlmock.MonitorPingsOption(true),
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
	)
	require.NoError(t, err)
	wrapper := formanSql.NewDB(db)
	msgMarshallerMock := mockMessage.NewMockMarshaller(ctrl)

	mock.ExpectBegin()
	mock.ExpectExec("create table if not exists saga ( uid varchar(255) not null primary key, parent_uid varchar(255) null, name varchar(255) null, payload text null, status varchar(255) null, started_at timestamp null, updated_at timestamp null, last_failed_ev text null );").
		WithArgs().
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("create table if not exists saga_history ( uid varchar(255) not null primary key, saga_uid varchar(255) not null, name varchar(255) null, status varchar(255) null, payload text null, origin varchar(255) null, created_at timestamp null, trace_uid varchar(255) null, constraint saga_history_saga_model_id_fk foreign key (saga_uid) references saga (uid) on update cascade on delete cascade );").
		WithArgs().
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	s, err := NewSQLSagaStore(wrapper, provider, msgMarshallerMock)
	require.NoError(t, err)

	return s, mock, msgMarshallerMock
}

type SagaExample struct {
	BaseSaga
	Data string
}

func (s *SagaExample) Init() {
	//TODO implement me
	panic("implement me")
}

func (s *SagaExample) Start(sagaCtx SagaContext) error {
	//TODO implement me
	panic("implement me")
}

func (s *SagaExample) Compensate(sagaCtx SagaContext) error {
	//TODO implement me
	panic("implement me")
}

func (s *SagaExample) Recover(sagaCtx SagaContext) error {
	//TODO implement me
	panic("implement me")
}

type ExampleEv struct {
	message.ObjectMeta
	Data string
}
