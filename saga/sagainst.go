package saga

import (
	"github.com/go-foreman/foreman/pubsub/message"
	"github.com/google/uuid"
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
	UID() string
	Saga() Saga
	Status() Status

	Start(sagaCtx SagaContext) error
	Compensate(sagaCtx SagaContext) error
	Recover(sagaCtx SagaContext) error
	Progress()
	Complete()
	Fail(ev message.Object)

	HistoryEvents() []HistoryEvent
	AttachEvent(ev message.Object, opts... AttachEvOpt)

	StartedAt() *time.Time
	UpdatedAt() *time.Time
	ParentID() string
}

type Status interface {
	InProgress() bool
	Failed() bool
	FailedOnEvent() message.Object
	Recovering() bool
	Compensating() bool
	Completed() bool
	String() string
}

func NewSagaInstance(id, parentId string, saga Saga) Instance {
	return &sagaInstance{
		uid:      id,
		parentID: parentId,
		saga:     saga,
		instanceStatus: instanceStatus{
			status:        sagaStatusCreated,
		},
		historyEvents: make([]HistoryEvent, 0),
	}
}

type sagaInstance struct {
	uid            string
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

func (s sagaInstance) UID() string {
	return s.uid
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

func (s *sagaInstance) Progress() {
	s.instanceStatus.status = sagaStatusInProgress
}

func (s *sagaInstance) Fail(ev message.Object) {
	s.instanceStatus.status = sagaStatusFailed
	s.instanceStatus.lastFailedEv = ev
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
	currentTime := time.Now().Round(time.Second).UTC()
	s.updatedAt = &currentTime
}

func (s *sagaInstance) AttachEvent(ev message.Object, opts... AttachEvOpt) {
	attachOpts := &attachEvOpts{}
	if len(opts) > 0 {
		for _, o := range opts {
			o(attachOpts)
		}
	}

 	historyEv := HistoryEvent{
		UID:          uuid.New().String(),
		CreatedAt:    time.Now().Round(time.Second).UTC(),
		Payload:      ev,
		OriginSource: attachOpts.origin,
		SagaStatus:   s.instanceStatus.status.String(),
		TraceUID: attachOpts.traceUID,
	}
	s.historyEvents = append(s.historyEvents, historyEv)
}

type HistoryEvent struct {
	UID          string      `json:"uid"`
	CreatedAt    time.Time   `json:"created_at"`
	Payload      message.Object `json:"payload"`
	OriginSource string      `json:"origin"`
	SagaStatus   string      `json:"saga_status"` //saga status at the moment
	TraceUID     string      `json:"trace_uid"` //uid of received message, could be empty
}

type attachEvOpts struct {
	traceUID string
	origin string
}

type AttachEvOpt func (o *attachEvOpts)

func WithTraceUID(uid string) AttachEvOpt {
	return func(o *attachEvOpts) {
		o.traceUID = uid
	}
}

func WithOrigin(origin string) AttachEvOpt {
	return func(o *attachEvOpts) {
		o.origin = origin
	}
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
	lastFailedEv message.Object
}

func (i instanceStatus) FailedOnEvent() message.Object {
	return i.lastFailedEv
}
