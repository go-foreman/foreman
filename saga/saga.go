package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	"github.com/pkg/errors"
	"time"
)

const (
	sagaStatusInProgress   status = "in_progress"
	sagaStatusFailed       status = "failed"
	sagaStatusCompleted    status = "completed"
	sagaStatusCreated      status = "created"
	sagaStatusCompensating status = "compensating"
	sagaStatusRecovering   status = "recovering"
)

type Instance interface {
	ID() string
	Saga() Saga

	Status() Status

	Start(sagaCtx SagaContext) error
	Compensate(sagaCtx SagaContext) error
	Recover(sagaCtx SagaContext) error
	Complete()
	Fail()

	HistoryEvents() []HistoryEvent
	AttachEvent(event HistoryEvent)

	StartedAt() time.Time
	UpdatedAt() time.Time
	ParentID() string
}

type Status interface {
	InProgress() bool
	Failed() bool
	Recovering() bool
	Compensating() bool
	Completed() bool
	String() string
}

func NewSagaInstance(id, parentId string, saga Saga) Instance {
	return &sagaInstance{id: id, parentID: parentId, saga: saga, status: sagaStatusCreated, startedAt: time.Now(), updatedAt: time.Now()}
}

type sagaInstance struct {
	id            string
	parentID      string
	saga          Saga
	historyEvents []HistoryEvent
	startedAt     time.Time
	updatedAt     time.Time
	status        Status
}

func (s sagaInstance) ParentID() string {
	return s.parentID
}

func (s sagaInstance) ID() string {
	return s.id
}

func (s sagaInstance) Saga() Saga {
	return s.saga
}

func (s sagaInstance) Status() Status {
	return s.status
}

func (s *sagaInstance) Start(sagaCtx SagaContext) error {
	s.status = sagaStatusInProgress
	s.update()
	return s.saga.Start(sagaCtx)
}

func (s *sagaInstance) Compensate(sagaCtx SagaContext) error {
	s.status = sagaStatusCompensating
	s.update()
	return s.saga.Compensate(sagaCtx)
}

func (s *sagaInstance) Recover(sagaCtx SagaContext) error {
	s.status = sagaStatusRecovering
	s.update()
	return s.saga.Recover(sagaCtx)
}

func (s *sagaInstance) Complete() {
	s.status = sagaStatusCompleted
	s.update()
}

func (s *sagaInstance) Fail() {
	s.status = sagaStatusFailed
	s.update()
}

func (s sagaInstance) HistoryEvents() []HistoryEvent {
	return s.historyEvents
}

func (s sagaInstance) StartedAt() time.Time {
	return s.startedAt
}

func (s sagaInstance) UpdatedAt() time.Time {
	return s.updatedAt
}

func (s *sagaInstance) update() {
	s.updatedAt = time.Now()
}

func (s *sagaInstance) AttachEvent(event HistoryEvent) {
	s.historyEvents = append(s.historyEvents, event)
}

func StatusFromStr(str string) (Status, error) {
	statuses := []status{sagaStatusInProgress, sagaStatusFailed, sagaStatusInProgress, sagaStatusCompensating, sagaStatusCompleted}
	for _, s := range statuses {
		if string(s) == str {
			return s, nil
		}
	}

	return nil, errors.Errorf("Unknown status string")
}

type status string

func (s status) InProgress() bool {
	return s == sagaStatusInProgress
}

func (s status) Failed() bool {
	return s == sagaStatusFailed
}

func (s status) Recovering() bool {
	return s == sagaStatusInProgress
}

func (s status) Compensating() bool {
	return s == sagaStatusCompensating
}

func (s status) Completed() bool {
	return s == sagaStatusCompleted
}

func (s status) String() string {
	return string(s)
}

type HistoryEvent struct {
	message.Metadata
	CreatedAt    time.Time   `json:"created_at"`
	Payload      interface{} `json:"payload"`
	OriginSource string      `json:"origin_source"`
	SagaStatus   string      `json:"saga_status"` //saga status at the moment
	Description  string      `json:"description"`
}

type Saga interface {
	//you should register all the handlers here
	Init()
	Start(execCtx SagaContext) error
	Compensate(execCtx SagaContext) error
	Recover(execCtx SagaContext) error
	EventHandlers() map[string]Executor
}

type BaseSaga struct {
	adjacencyMap map[string]Executor
}

type Executor func(execCtx SagaContext) error

func (b *BaseSaga) AddEventHandler(event interface{}, handler Executor) *BaseSaga {
	//lazy initialization
	if b.adjacencyMap == nil {
		b.adjacencyMap = make(map[string]Executor)
	}

	eventKey := scheme.WithStruct(event)

	b.adjacencyMap[eventKey()] = handler
	return b
}

func (b BaseSaga) EventHandlers() map[string]Executor {
	return b.adjacencyMap
}
