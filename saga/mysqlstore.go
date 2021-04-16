package saga

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/pkg/errors"
)

type mysqlStore struct {
	msgMarshaller message.Marshaller
	db            *sql.DB
}

func NewMysqlSagaStore(db *sql.DB, msgMarshaller message.Marshaller) (Store, error) {
	err := initMysqlTables(db)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &mysqlStore{db: db, msgMarshaller: msgMarshaller}, nil
}

// Create saves saga instance into mysql store. History events, last failed event are not persisted at this step, there is not way for them to be at creation step.
func (s mysqlStore) Create(ctx context.Context, sagaInstance Instance) error {
	payload, err := s.msgMarshaller.Marshal(sagaInstance.Saga())

	if err != nil {
		return errors.WithStack(err)
	}

	sagaName := sagaInstance.Saga().GroupKind().String()

	tx, err := s.db.Begin()

	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %v (uid, parent_uid, name, payload, status, started_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?);", sagaTableName),
		sagaInstance.UID(),
		sagaInstance.ParentID(),
		sagaName,
		payload,
		sagaInstance.Status().String(),
		sagaInstance.StartedAt(),
		sagaInstance.UpdatedAt(),
	)
	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "commiting saga instance %s into mysql store", sagaInstance.UID())
	}

	return nil
}

func (s mysqlStore) Update(ctx context.Context, sagaInstance Instance) error {
	payload, err := s.msgMarshaller.Marshal(sagaInstance.Saga())
	sagaName := sagaInstance.Saga().GroupKind().String()

	if err != nil {
		return errors.Wrapf(err, "marshaling saga instance %s on update", sagaInstance.UID())
	}

	var lastFailedEv []byte

	if sagaInstance.Status().FailedOnEvent() != nil {
		var err error
		lastFailedEv, err = s.msgMarshaller.Marshal(sagaInstance.Status().FailedOnEvent())

		if err != nil {
			return errors.Wrapf(err, "marshaling last failed event of saga instance %s on update", sagaInstance.UID())
		}
	}

	tx, err := s.db.Begin()

	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf("UPDATE %v SET parent_uid=?, name=?, payload=?, status=?, started_at=?, updated_at=?, last_failed_ev=? WHERE uid=?;", sagaTableName),
		sagaInstance.ParentID(),
		sagaName,
		payload,
		sagaInstance.Status().String(),
		sagaInstance.StartedAt(),
		sagaInstance.UpdatedAt(),
		lastFailedEv,
		sagaInstance.UID(),
	)

	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.WithStack(err)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT uid FROM %v WHERE saga_uid=?;", sagaHistoryTableName), sagaInstance.UID())

	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.Wrapf(err, "querying %s for saga_uid %s", sagaHistoryTableName, sagaInstance.UID())
	}

	var id string
	ids := make(map[string]string)
	for rows.Next() {
		err := rows.Scan(&id)

		if err != nil {
			if rErr := tx.Rollback(); rErr != nil {
				return errors.Wrapf(rErr, "error rollback when %s", err)
			}
			return errors.WithStack(err)
		}

		ids[id] = id
	}

	if len(ids) < len(sagaInstance.HistoryEvents()) {
		for _, ev := range sagaInstance.HistoryEvents() {
			if _, exists := ids[ev.UID]; exists {
				continue
			}

			payload, err := s.msgMarshaller.Marshal(ev.Payload)

			if err != nil {
				if rErr := tx.Rollback(); rErr != nil {
					return errors.Wrapf(rErr, "error rollback when %s", err)
				}

				return errors.WithStack(err)
			}

			_, err = tx.Exec(fmt.Sprintf("INSERT INTO %v (uid, saga_uid, name, status, payload, origin, created_at, trace_uid) VALUES (?, ?, ?, ?, ?, ?, ?, ?);", sagaHistoryTableName),
				ev.UID,
				sagaInstance.UID(),
				ev.Payload.GroupKind().String(),
				ev.SagaStatus,
				payload,
				ev.OriginSource,
				ev.CreatedAt,
				ev.TraceUID,
			)

			if err != nil {
				if rErr := tx.Rollback(); rErr != nil {
					return errors.Wrapf(rErr, "error rollback when %s", err)
				}
				return errors.Wrapf(err, "inserting history event %v for saga %s", ev, sagaInstance.UID())
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s mysqlStore) GetById(ctx context.Context, sagaId string) (Instance, error) {
	sagaData := sagaSqlModel{}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT s.uid, s.parent_uid, s.name, s.payload, s.status, s.last_failed_ev, s.started_at, s.updated_at FROM %v s WHERE uid=?;", sagaTableName), sagaId).
		Scan(
			&sagaData.ID,
			&sagaData.ParentID,
			&sagaData.Name,
			&sagaData.Payload,
			&sagaData.Status,
			&sagaData.LastFailedMsg,
			&sagaData.StartedAt,
			&sagaData.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}

	sagaInstance, err := s.instanceFromModel(sagaData)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	messages, err := s.queryEvents(sagaId)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	sagaInstance.historyEvents = messages

	return sagaInstance, nil
}

func (s mysqlStore) GetByFilter(ctx context.Context, filters... FilterOption) ([]Instance, error) {
	if len(filters) == 0 {
		return nil, errors.Errorf("No filters found, you have to specify at least one so result won't be whole store")
	}

	opts := &filterOptions{}

	for _, filter := range filters {
		filter(opts)
	}

	//todo use https://github.com/Masterminds/squirrel ? +1 dependency, is it really needed?
	query := fmt.Sprintf(`SELECT s.uid, s.parent_uid, s.name, s.payload, s.status, s.started_at, s.updated_at, s.last_failed_ev, sh.uid, sh.name, sh.status, sh.payload, sh.origin, sh.created_at, sh.trace_uid FROM %s s LEFT JOIN %s sh ON s.uid = sh.saga_uid WHERE`, sagaTableName, sagaHistoryTableName)

	var (
		args       []interface{}
		conditions []string
	)

	if opts.sagaId != "" {
		conditions = append(conditions, " s.uid = ?")
		args = append(args, opts.sagaId)
	}

	if opts.status != "" {
		conditions = append(conditions, " s.status = ?")
		args = append(args, opts.status)
	}

	if opts.sagaName != "" {
		conditions = append(conditions, " s.name = ?")
		args = append(args, opts.sagaName)
	}

	if len(conditions) == 0 {
		return nil, errors.Errorf("All specified filters are empty, you have to specify at least one so result won't be whole store")
	}

	for i, condition := range conditions {
		query += condition

		if i < len(conditions)-1 {
			query += " AND"
		}

		if i == len(conditions)-1 {
			query += ";"
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, errors.Wrap(err, "querying sagas with filter")
	}

	sagas := make(map[string]*sagaInstance)

	for rows.Next() {
		sagaData := sagaSqlModel{}
		ev := historyEventSqlModel{}

		if err := rows.Scan(
			&sagaData.ID,
			&sagaData.ParentID,
			&sagaData.Name,
			&sagaData.Payload,
			&sagaData.Status,
			&sagaData.StartedAt,
			&sagaData.UpdatedAt,
			&sagaData.LastFailedMsg,
			&ev.ID,
			&ev.Name,
			&ev.SagaStatus,
			&ev.Payload,
			&ev.OriginSource,
			&ev.CreatedAt,
			&ev.TraceUID,
		); err != nil {
			return nil, errors.WithStack(err)
		}

		sagaInstance, exists := sagas[sagaData.ID.String]

		if !exists {
			instance, err := s.instanceFromModel(sagaData)

			if err != nil {
				return nil, errors.WithStack(err)
			}
			sagas[sagaData.ID.String] = instance
			sagaInstance = instance
		}

		if ev.ID.String != "" {
			historyEvent, err := s.eventFromModel(ev)

			if err != nil {
				return nil, errors.WithStack(err)
			}

			sagaInstance.historyEvents = append(sagaInstance.historyEvents, *historyEvent)
		}
	}

	if rows.Err() != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]Instance, len(sagas))

	var i int
	for _, instance := range sagas {
		res[i] = instance
		i++
	}

	return res, nil
}

func (s mysqlStore) Delete(ctx context.Context, sagaId string) error {
	res, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %v WHERE uid=?;", sagaTableName), sagaId)
	if err != nil {
		return errors.Wrapf(err, "executing delete query for saga %s", sagaId)
	}

	rows, err := res.RowsAffected()

	if err != nil {
		return errors.Wrapf(err, "getting response of  delete query for saga %s", sagaId)
	}

	if rows > 0 {
		return nil
	}

	return errors.Errorf("no saga instance %s found", sagaId)
}

func (s mysqlStore) queryEvents(sagaId string) ([]HistoryEvent, error) {
	rows, err := s.db.Query(fmt.Sprintf("SELECT uid, name, status, payload, origin, created_at, trace_uid FROM %v WHERE saga_uid=? ORDER BY created_at;", sagaHistoryTableName), sagaId)

	if err != nil {
		return nil, errors.Wrapf(err, "querying events for saga %s", sagaId)
	}

	messages := make([]HistoryEvent, 0)

	for rows.Next() {
		ev := historyEventSqlModel{}

		if err := rows.Scan(
			&ev.ID,
			&ev.Name,
			&ev.SagaStatus,
			&ev.Payload,
			&ev.OriginSource,
			&ev.CreatedAt,
			&ev.TraceUID,
		); err != nil {
			return nil, errors.Wrapf(err, "scanning events for saga %s", sagaId)
		}

		hEv, err := s.eventFromModel(ev)

		if err != nil {
			return nil, errors.WithStack(err)
		}

		messages = append(messages, *hEv)
	}

	if rows.Err() != nil {
		return nil, errors.WithStack(err)
	}

	return messages, nil
}

func (s mysqlStore) eventFromModel(ev historyEventSqlModel) (*HistoryEvent, error) {
	eventPayload, err := s.msgMarshaller.Unmarshal(ev.Payload)

	if err != nil {
		return nil, errors.Wrapf(err, "error deserializing payload into event %s", ev.ID.String)
	}

	res := &HistoryEvent{
		UID: ev.ID.String,
		SagaStatus: ev.SagaStatus.String,
		Payload: eventPayload,
		CreatedAt: ev.CreatedAt.Time,
		OriginSource: ev.OriginSource.String,
		TraceUID: ev.TraceUID.String,
	}

	return res, nil
}

func (s mysqlStore) instanceFromModel(sagaData sagaSqlModel) (*sagaInstance, error) {
	status, err := statusFromStr(sagaData.Status.String)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing status of %s", sagaData.ID.String)
	}

	sagaInstance := &sagaInstance{
		uid: sagaData.ID.String,
		instanceStatus:    instanceStatus{
			status:        status,
		},
		parentID:  sagaData.ParentID.String,
		historyEvents: make([]HistoryEvent, 0),
	}

	if sagaData.StartedAt.Valid {
		sagaInstance.startedAt = &sagaData.StartedAt.Time
	}

	if sagaData.UpdatedAt.Valid {
		sagaInstance.updatedAt = &sagaData.UpdatedAt.Time
	}

	if len(sagaData.LastFailedMsg) > 0 {
		sagaInstance.instanceStatus.lastFailedEv, err = s.msgMarshaller.Unmarshal(sagaData.LastFailedMsg)
		if err != nil {
			return nil, errors.Wrapf(err, "unmarshaling last failed ev %v for saga %s", sagaData.LastFailedMsg, sagaData.ID.String)
		}
	}

	saga, err := s.msgMarshaller.Unmarshal(sagaData.Payload)

	if err != nil {
		return nil, errors.Wrapf(err, "error deserializing payload %v into saga %s", sagaData.Payload, sagaData.Name.String)
	}

	sagaInterface, ok := saga.(Saga)

	if !ok {
		return nil, errors.New("error converting %s into type Saga interface")
	}

	sagaInstance.saga = sagaInterface

	return sagaInstance, nil
}

func initMysqlTables(db *sql.DB) error {
	tx, err := db.Begin()

	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.Exec(fmt.Sprintf(`create table if not exists %v
	(
		uid varchar(255) not null primary key,
		parent_uid varchar(255) null,
		name varchar(255) null,
		payload text null,
		status varchar(255) null,
		started_at timestamp null,
		updated_at timestamp null,
		last_failed_ev text null
	);`, sagaTableName))

	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.WithStack(err)
	}

	_, err = tx.Exec(fmt.Sprintf(`create table if not exists %v
	(
		uid varchar(255) not null primary key,
		saga_uid varchar(255) not null,
		name varchar(255) null,
		status varchar(255) null,
		payload text null,
		origin varchar(255) null,
		created_at timestamp null,
		trace_uid varchar(255) null,
		constraint saga_history_saga_model_id_fk
			foreign key (saga_uid) references %v (uid)
				on update cascade on delete cascade
	);`, sagaHistoryTableName, sagaTableName))

	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

type sagaSqlModel struct {
	ID        sql.NullString
	ParentID  sql.NullString
	Name      sql.NullString
	Payload   []byte
	Status    sql.NullString
	LastFailedMsg []byte
	StartedAt sql.NullTime
	UpdatedAt sql.NullTime
}

type historyEventSqlModel struct {
	ID      sql.NullString
	Name    sql.NullString
	CreatedAt    sql.NullTime
	Payload      []byte
	OriginSource sql.NullString
	SagaStatus   sql.NullString
	TraceUID     sql.NullString
}
