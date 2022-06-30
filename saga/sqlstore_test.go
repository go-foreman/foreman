package saga

import (
	"context"
	"testing"

	"github.com/go-foreman/foreman/runtime/scheme"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/saga/sql"
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
		wrapper := sql.NewDB(db)

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
		wrapper := sql.NewDB(db)

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
		wrapper := sql.NewDB(db)

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
		sagaInstance.AddHistoryEvent(&ExampleEv{Data: "data"}, &AddHistoryEvent{
			TraceUID: "xxx",
			Origin:   "yyy",
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

		require.Len(t, sagaInstance.HistoryEvents(), 1)
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
			WillReturnRows(sqlmock.NewRows([]string{"uid"}))

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
}

func createStore(t *testing.T, ctrl *gomock.Controller, provider SQLDriver) (Store, sqlmock.Sqlmock, *mockMessage.MockMarshaller) {
	db, mock, err := sqlmock.New(
		sqlmock.MonitorPingsOption(true),
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual),
	)
	require.NoError(t, err)
	wrapper := sql.NewDB(db)
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
