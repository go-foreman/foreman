package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/go-foreman/foreman/runtime/scheme"
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

	StartedAt() *time.Time
	UpdatedAt() *time.Time
	ParentID() string
}

type Status interface {
	InProgress() bool
	Failed() bool
	FailedOnEvent() interface{}
	Recovering() bool
	Compensating() bool
	Completed() bool
	String() string
}

func NewSagaInstance(id, parentId string, saga Saga) Instance {
	return &sagaInstance{
		id: id,
		parentID: parentId,
		saga: saga,
		instanceStatus: instanceStatus{
			status:        sagaStatusCreated,
		},
		historyEvents: make([]HistoryEvent, 0),
	}
}

type sagaInstance struct {
	id             string
	parentID       string
	saga           Saga
	historyEvents  []HistoryEvent
	startedAt      *time.Time
	updatedAt      *time.Time
	instanceStatus instanceStatus
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
	return s.instanceStatus
}

func (s *sagaInstance) Start(sagaCtx SagaContext) error {
	s.instanceStatus.status = sagaStatusInProgress
	current := time.Now().Round(time.Second).UTC()
	s.startedAt = &current
	s.update()
	return s.saga.Start(sagaCtx)
}

func (s *sagaInstance) Compensate(sagaCtx SagaContext) error {
	s.instanceStatus.status = sagaStatusCompensating
	s.update()
	return s.saga.Compensate(sagaCtx)
}

func (s *sagaInstance) Recover(sagaCtx SagaContext) error {
	s.instanceStatus.status = sagaStatusRecovering
	s.update()
	return s.saga.Recover(sagaCtx)
}

func (s *sagaInstance) Complete() {
	s.instanceStatus.status = sagaStatusCompleted
	s.update()
}

func (s *sagaInstance) Fail() {
	s.instanceStatus.status = sagaStatusFailed
	s.update()
}

func (s sagaInstance) HistoryEvents() []HistoryEvent {
	return s.historyEvents
}

func (s sagaInstance) StartedAt() *time.Time {
	if s.startedAt != nil {
		return s.startedAt
	}
	return nil
}

func (s sagaInstance) UpdatedAt() *time.Time {
	if s.updatedAt != nil {
		return s.updatedAt
	}
	return s.updatedAt
}

func (s *sagaInstance) update() {
	currentTime := time.Now()
	s.updatedAt = &currentTime
}

func (s *sagaInstance) AttachEvent(event HistoryEvent) {
	s.historyEvents = append(s.historyEvents, event)
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

type instanceStatus struct {
	status
	lastFailedEv interface{}
}

func (i instanceStatus) FailedOnEvent() interface{} {
	return i.lastFailedEv
}

type HistoryEvent struct {
	UID          string      `json:"uid"`
	CreatedAt    time.Time   `json:"created_at"`
	Payload      interface{} `json:"payload"`
	OriginSource string      `json:"origin_source"`
	SagaStatus   string      `json:"saga_status"` //saga status at the moment
	Description  string      `json:"description"`
}

type Saga interface {
	Init()
	Start(execCtx SagaContext) error
	Compensate(execCtx SagaContext) error
	Recover(execCtx SagaContext) error
	EventHandlers() map[scheme.GroupKind]Executor
}

type BaseSaga struct {
	adjacencyMap map[scheme.GroupKind]Executor
}

type Executor func(execCtx SagaContext) error

func (b *BaseSaga) AddEventHandler(ev message.Object, handler Executor) *BaseSaga {
	//lazy initialization
	if b.adjacencyMap == nil {
		b.adjacencyMap = make(map[scheme.GroupKind]Executor)
	}

	b.adjacencyMap[ev.GroupKind()] = handler
	return b
}

func (b BaseSaga) EventHandlers() map[scheme.GroupKind]Executor {
	return b.adjacencyMap
}
