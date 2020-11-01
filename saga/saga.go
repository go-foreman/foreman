package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
	"time"
)

const (
	sagaStatusInProgress   = "in_progress"
	sagaStatusFailed       = "failed"
	sagaStatusCompleted    = "completed"
	sagaStatusCreated      = "created"
	sagaStatusCompensating = "compensating"
	sagaStatusRecovering   = "recovering"
)

type Instance interface {
	ID() string
	Saga() Saga
	Status() string

	Start(sagaCtx SagaContext) error
	InProgress() bool

	Compensate(sagaCtx SagaContext) error
	Compensating() bool

	Recover(sagaCtx SagaContext) error
	Recovering() bool

	Completed() bool
	Complete()

	Failed() bool
	Fail()

	HistoryEvents() []HistoryEvent
	AttachEvent(event HistoryEvent)

	StartedAt() time.Time
	UpdatedAt() time.Time
	ParentID() string
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
	status        string
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

func (s sagaInstance) Status() string {
	return s.status
}

func (s *sagaInstance) Start(sagaCtx SagaContext) error {
	s.status = sagaStatusInProgress
	s.update()
	return s.saga.Start(sagaCtx)
}

func (s sagaInstance) InProgress() bool {
	return s.status == sagaStatusInProgress
}

func (s *sagaInstance) Compensate(sagaCtx SagaContext) error {
	s.status = sagaStatusCompensating
	s.update()
	return s.saga.Compensate(sagaCtx)
}

func (s sagaInstance) Compensating() bool {
	return s.status == sagaStatusCompensating
}

func (s *sagaInstance) Recover(sagaCtx SagaContext) error {
	s.status = sagaStatusRecovering
	s.update()
	return s.saga.Recover(sagaCtx)
}

func (s sagaInstance) Recovering() bool {
	return s.status == sagaStatusRecovering
}

func (s *sagaInstance) Complete() {
	s.status = sagaStatusCompleted
	s.update()
}

func (s sagaInstance) Completed() bool {
	return s.status == sagaStatusCompleted
}

func (s *sagaInstance) Fail() {
	s.status = sagaStatusFailed
	s.update()
}

func (s sagaInstance) Failed() bool {
	return s.status == sagaStatusFailed
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
