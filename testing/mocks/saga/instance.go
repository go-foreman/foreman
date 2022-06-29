// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/go-foreman/foreman/saga (interfaces: Instance)

// Package saga is a generated GoMock package.
package saga

import (
	reflect "reflect"
	time "time"

	message "github.com/go-foreman/foreman/pubsub/message"
	saga "github.com/go-foreman/foreman/saga"
	gomock "github.com/golang/mock/gomock"
)

// MockInstance is a mock of Instance interface.
type MockInstance struct {
	ctrl     *gomock.Controller
	recorder *MockInstanceMockRecorder
}

// MockInstanceMockRecorder is the mock recorder for MockInstance.
type MockInstanceMockRecorder struct {
	mock *MockInstance
}

// NewMockInstance creates a new mock instance.
func NewMockInstance(ctrl *gomock.Controller) *MockInstance {
	mock := &MockInstance{ctrl: ctrl}
	mock.recorder = &MockInstanceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInstance) EXPECT() *MockInstanceMockRecorder {
	return m.recorder
}

// AddHistoryEvent mocks base method.
func (m *MockInstance) AddHistoryEvent(arg0 message.Object, arg1 ...saga.AddEvOpt) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "AddHistoryEvent", varargs...)
}

// AddHistoryEvent indicates an expected call of AddHistoryEvent.
func (mr *MockInstanceMockRecorder) AddHistoryEvent(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddHistoryEvent", reflect.TypeOf((*MockInstance)(nil).AddHistoryEvent), varargs...)
}

// Compensate mocks base method.
func (m *MockInstance) Compensate(arg0 saga.SagaContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Compensate", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Compensate indicates an expected call of Compensate.
func (mr *MockInstanceMockRecorder) Compensate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Compensate", reflect.TypeOf((*MockInstance)(nil).Compensate), arg0)
}

// Complete mocks base method.
func (m *MockInstance) Complete() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Complete")
}

// Complete indicates an expected call of Complete.
func (mr *MockInstanceMockRecorder) Complete() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Complete", reflect.TypeOf((*MockInstance)(nil).Complete))
}

// Fail mocks base method.
func (m *MockInstance) Fail(arg0 message.Object) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Fail", arg0)
}

// Fail indicates an expected call of Fail.
func (mr *MockInstanceMockRecorder) Fail(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fail", reflect.TypeOf((*MockInstance)(nil).Fail), arg0)
}

// HistoryEvents mocks base method.
func (m *MockInstance) HistoryEvents() []saga.HistoryEvent {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "HistoryEvents")
	ret0, _ := ret[0].([]saga.HistoryEvent)
	return ret0
}

// HistoryEvents indicates an expected call of HistoryEvents.
func (mr *MockInstanceMockRecorder) HistoryEvents() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "HistoryEvents", reflect.TypeOf((*MockInstance)(nil).HistoryEvents))
}

// ParentID mocks base method.
func (m *MockInstance) ParentID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ParentID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ParentID indicates an expected call of ParentID.
func (mr *MockInstanceMockRecorder) ParentID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParentID", reflect.TypeOf((*MockInstance)(nil).ParentID))
}

// Progress mocks base method.
func (m *MockInstance) Progress() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Progress")
}

// Progress indicates an expected call of Progress.
func (mr *MockInstanceMockRecorder) Progress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Progress", reflect.TypeOf((*MockInstance)(nil).Progress))
}

// Recover mocks base method.
func (m *MockInstance) Recover(arg0 saga.SagaContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recover", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Recover indicates an expected call of Recover.
func (mr *MockInstanceMockRecorder) Recover(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recover", reflect.TypeOf((*MockInstance)(nil).Recover), arg0)
}

// Saga mocks base method.
func (m *MockInstance) Saga() saga.Saga {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Saga")
	ret0, _ := ret[0].(saga.Saga)
	return ret0
}

// Saga indicates an expected call of Saga.
func (mr *MockInstanceMockRecorder) Saga() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Saga", reflect.TypeOf((*MockInstance)(nil).Saga))
}

// Start mocks base method.
func (m *MockInstance) Start(arg0 saga.SagaContext) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockInstanceMockRecorder) Start(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockInstance)(nil).Start), arg0)
}

// StartedAt mocks base method.
func (m *MockInstance) StartedAt() *time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartedAt")
	ret0, _ := ret[0].(*time.Time)
	return ret0
}

// StartedAt indicates an expected call of StartedAt.
func (mr *MockInstanceMockRecorder) StartedAt() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartedAt", reflect.TypeOf((*MockInstance)(nil).StartedAt))
}

// Status mocks base method.
func (m *MockInstance) Status() saga.Status {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status")
	ret0, _ := ret[0].(saga.Status)
	return ret0
}

// Status indicates an expected call of Status.
func (mr *MockInstanceMockRecorder) Status() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockInstance)(nil).Status))
}

// UID mocks base method.
func (m *MockInstance) UID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UID")
	ret0, _ := ret[0].(string)
	return ret0
}

// UID indicates an expected call of UID.
func (mr *MockInstanceMockRecorder) UID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UID", reflect.TypeOf((*MockInstance)(nil).UID))
}

// UpdatedAt mocks base method.
func (m *MockInstance) UpdatedAt() *time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdatedAt")
	ret0, _ := ret[0].(*time.Time)
	return ret0
}

// UpdatedAt indicates an expected call of UpdatedAt.
func (mr *MockInstanceMockRecorder) UpdatedAt() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdatedAt", reflect.TypeOf((*MockInstance)(nil).UpdatedAt))
}
