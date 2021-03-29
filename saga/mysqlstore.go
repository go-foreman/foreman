package saga

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/pkg/errors"
)

type mysqlStore struct {
	typesRegistry scheme.KnownTypesRegistry
	db            *sql.DB
}

func NewMysqlSagaStore(db *sql.DB, registry scheme.KnownTypesRegistry) (Store, error) {
	err := initMysqlTables(db)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &mysqlStore{db: db, typesRegistry: registry}, nil
}

//History events are not persisted at this step
func (s mysqlStore) Create(ctx context.Context, sagaInstance Instance) error {

	payload, err := json.Marshal(sagaInstance.Saga())

	if err != nil {
		return errors.WithStack(err)
	}

	sagaName := scheme.WithStruct(sagaInstance.Saga())()

	data := &sagaModel{
		ID:        sagaInstance.ID(),
		ParentID:  sagaInstance.ParentID(),
		Name:      sagaName,
		Payload:   payload,
		Status:    sagaInstance.Status().String(),
		StartedAt: sagaInstance.StartedAt(),
		UpdatedAt: sagaInstance.UpdatedAt(),
	}

	tx, err := s.db.Begin()

	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf("INSERT INTO %v VALUES (?, ?, ?, ?, ?, ?, ?);", sagaTableName),
		data.ID,
		data.ParentID,
		data.Name,
		data.Payload,
		data.Status,
		data.StartedAt,
		data.UpdatedAt)
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

func (s mysqlStore) Update(ctx context.Context, sagaInstance Instance) error {
	payload, err := json.Marshal(sagaInstance.Saga())

	if err != nil {
		return errors.WithStack(err)
	}

	sagaName := scheme.WithStruct(sagaInstance.Saga())()

	sagaData := &sagaModel{
		ID:        sagaInstance.ID(),
		ParentID:  sagaInstance.ParentID(),
		Name:      sagaName,
		Payload:   payload,
		Status:    sagaInstance.Status().String(),
		StartedAt: sagaInstance.StartedAt(),
		UpdatedAt: sagaInstance.UpdatedAt(),
	}

	tx, err := s.db.Begin()

	if err != nil {
		return errors.WithStack(err)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf("UPDATE %v SET parent_id=?, name=?, payload=?, status=?, started_at=?, updated_at=? WHERE id=?;", sagaTableName),
		sagaData.ParentID,
		sagaData.Name,
		sagaData.Payload,
		sagaData.Status,
		sagaData.StartedAt,
		sagaData.UpdatedAt,
		sagaData.ID)

	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.WithStack(err)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT id FROM %v WHERE saga_id=?;", sagaHistoryTableName), sagaData.ID)

	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.WithStack(err)
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
		for _, m := range sagaInstance.HistoryEvents() {
			if _, exists := ids[m.ID]; exists {
				continue
			}

			payload, err := json.Marshal(m.Payload)

			if err != nil {
				if rErr := tx.Rollback(); rErr != nil {
					return errors.Wrapf(rErr, "error rollback when %s", err)
				}

				return errors.WithStack(err)
			}

			_, err = tx.Exec(fmt.Sprintf("INSERT INTO %v VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);", sagaHistoryTableName),
				m.ID,
				sagaData.ID,
				m.Name,
				m.Type,
				m.SagaStatus,
				payload,
				m.Description,
				m.OriginSource,
				m.CreatedAt)
			if err != nil {
				if rErr := tx.Rollback(); rErr != nil {
					return errors.Wrapf(rErr, "error rollback when %s", err)
				}
				return errors.WithStack(err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s mysqlStore) GetById(ctx context.Context, sagaId string) (Instance, error) {
	sagaData := sagaModel{}
	err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT * FROM %v WHERE id=?;", sagaTableName), sagaId).
		Scan(
			&sagaData.ID,
			&sagaData.ParentID,
			&sagaData.Name,
			&sagaData.Payload,
			&sagaData.Status,
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
	query := fmt.Sprintf(`SELECT s.id, s.parent_id, s.name, s.payload, s.status, s.started_at, s.updated_at, sh.id, sh.name, sh.type, sh.status, sh.payload, description, sh.origin_source, sh.created_at FROM %s s LEFT JOIN %s sh ON s.id = sh.saga_id WHERE`, sagaTableName, sagaHistoryTableName)

	var (
		args       []interface{}
		conditions []string
	)

	if opts.sagaId != "" {
		conditions = append(conditions, " s.id = ?")
		args = append(args, opts.sagaId)
	}

	if opts.status != "" {
		conditions = append(conditions, " s.status = ?")
		args = append(args, opts.status)
	}

	if opts.sagaType != "" {
		conditions = append(conditions, " s.name = ?")
		args = append(args, opts.sagaType)
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
		return nil, errors.WithStack(err)
	}

	sagas := make(map[string]*sagaInstance)

	for rows.Next() {
		sagaData := sagaModel{}
		ev := historyEventModel{}

		if err := rows.Scan(
			&sagaData.ID,
			&sagaData.ParentID,
			&sagaData.Name,
			&sagaData.Payload,
			&sagaData.Status,
			&sagaData.StartedAt,
			&sagaData.UpdatedAt,
			&ev.ID,
			&ev.Name,
			&ev.Type,
			&ev.SagaStatus,
			&ev.Payload,
			&ev.Description,
			&ev.OriginSource,
			&ev.CreatedAt); err != nil {
			return nil, errors.WithStack(err)
		}

		sagaInstance, exists := sagas[sagaData.ID]

		if !exists {
			instance, err := s.instanceFromModel(sagaData)

			if err != nil {
				return nil, errors.WithStack(err)
			}
			sagas[sagaData.ID] = instance
			sagaInstance = instance
		}

		historyEvent, err := s.eventFromModel(ev)

		if err != nil {
			return nil, errors.WithStack(err)
		}

		sagaInstance.historyEvents = append(sagaInstance.historyEvents, *historyEvent)
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
	res, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %v WHERE id=?;", sagaTableName), sagaId)
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
	rows, err := s.db.Query(fmt.Sprintf("SELECT id, name, type, status, payload, description, origin_source, created_at FROM %v WHERE saga_id=? ORDER BY created_at;", sagaHistoryTableName), sagaId)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	messages := make([]HistoryEvent, 0)

	for rows.Next() {
		ev := historyEventModel{}

		if err := rows.Scan(
			&ev.ID,
			&ev.Name,
			&ev.Type,
			&ev.SagaStatus,
			&ev.Payload,
			&ev.Description,
			&ev.OriginSource,
			&ev.CreatedAt); err != nil {
			return nil, errors.WithStack(err)
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

func (s mysqlStore) eventFromModel(ev historyEventModel) (*HistoryEvent, error) {
	res := &HistoryEvent{}

	if err := json.Unmarshal(ev.Payload, &res.Payload); err != nil {
		return nil, errors.WithStack(err)
	}

	res.Metadata = ev.Metadata
	res.CreatedAt = ev.CreatedAt
	res.OriginSource = ev.OriginSource
	res.SagaStatus = ev.SagaStatus
	res.Description = ev.Description

	return res, nil
}

func (s mysqlStore) instanceFromModel(sagaData sagaModel) (*sagaInstance, error) {
	status, err := StatusFromStr(sagaData.Status)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing status of %s", sagaData.ID)
	}
	sagaInstance := &sagaInstance{
		id:        sagaData.ID,
		startedAt: sagaData.StartedAt,
		updatedAt: sagaData.UpdatedAt,
		status:    status,
		parentID:  sagaData.ParentID,
	}

	saga, err := s.typesRegistry.LoadType(scheme.WithKey(sagaData.Name))
	sagaType := s.typesRegistry.GetType(scheme.WithKey(sagaData.Name))

	if err != nil {
		return nil, errors.WithStack(err)
	}

	if err := json.Unmarshal(sagaData.Payload, saga); err != nil {
		return nil, errors.Errorf("error deserializing payload into saga of type %s ", sagaType.Kind().String())
	}

	sagaInterface, ok := saga.(Saga)

	if !ok {
		return nil, errors.New("Error converting %s into type Saga interface")
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
		id varchar(255) not null primary key,
		parent_id varchar(255) null,
		name varchar(255) null,
		payload text null,
		status varchar(255) null,
		started_at timestamp null,
		updated_at timestamp null
	);`, sagaTableName))

	if err != nil {
		if rErr := tx.Rollback(); rErr != nil {
			return errors.Wrapf(rErr, "error rollback when %s", err)
		}
		return errors.WithStack(err)
	}

	_, err = tx.Exec(fmt.Sprintf(`create table if not exists %v
	(
		id varchar(255) not null primary key,
		saga_id varchar(255) not null,
		name varchar(255) null,
		type varchar(255) null,
		status varchar(255) null,
		payload text null,
		description text null,
		origin_source varchar(255) null,
		created_at timestamp null,
		constraint saga_history_saga_model_id_fk
			foreign key (saga_id) references %v (id)
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
