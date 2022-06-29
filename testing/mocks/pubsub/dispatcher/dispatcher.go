// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/go-foreman/foreman/pubsub/dispatcher (interfaces: Dispatcher)

// Package dispatcher is a generated GoMock package.
package dispatcher

import (
	reflect "reflect"

	dispatcher "github.com/go-foreman/foreman/pubsub/dispatcher"
	message "github.com/go-foreman/foreman/pubsub/message"
	execution "github.com/go-foreman/foreman/pubsub/message/execution"
	gomock "github.com/golang/mock/gomock"
)

// MockDispatcher is a mock of Dispatcher interface.
type MockDispatcher struct {
	ctrl     *gomock.Controller
	recorder *MockDispatcherMockRecorder
}

// MockDispatcherMockRecorder is the mock recorder for MockDispatcher.
type MockDispatcherMockRecorder struct {
	mock *MockDispatcher
}

// NewMockDispatcher creates a new mock instance.
func NewMockDispatcher(ctrl *gomock.Controller) *MockDispatcher {
	mock := &MockDispatcher{ctrl: ctrl}
	mock.recorder = &MockDispatcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDispatcher) EXPECT() *MockDispatcherMockRecorder {
	return m.recorder
}

// Match mocks base method.
func (m *MockDispatcher) Match(arg0 message.Object) []execution.Executor {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Match", arg0)
	ret0, _ := ret[0].([]execution.Executor)
	return ret0
}

// Match indicates an expected call of Match.
func (mr *MockDispatcherMockRecorder) Match(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Match", reflect.TypeOf((*MockDispatcher)(nil).Match), arg0)
}

// SubscribeForAllEvents mocks base method.
func (m *MockDispatcher) SubscribeForAllEvents(arg0 execution.Executor) dispatcher.Dispatcher {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeForAllEvents", arg0)
	ret0, _ := ret[0].(dispatcher.Dispatcher)
	return ret0
}

// SubscribeForAllEvents indicates an expected call of SubscribeForAllEvents.
func (mr *MockDispatcherMockRecorder) SubscribeForAllEvents(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeForAllEvents", reflect.TypeOf((*MockDispatcher)(nil).SubscribeForAllEvents), arg0)
}

// SubscribeForCmd mocks base method.
func (m *MockDispatcher) SubscribeForCmd(arg0 message.Object, arg1 execution.Executor) dispatcher.Dispatcher {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeForCmd", arg0, arg1)
	ret0, _ := ret[0].(dispatcher.Dispatcher)
	return ret0
}

// SubscribeForCmd indicates an expected call of SubscribeForCmd.
func (mr *MockDispatcherMockRecorder) SubscribeForCmd(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeForCmd", reflect.TypeOf((*MockDispatcher)(nil).SubscribeForCmd), arg0, arg1)
}

// SubscribeForEvent mocks base method.
func (m *MockDispatcher) SubscribeForEvent(arg0 message.Object, arg1 execution.Executor) dispatcher.Dispatcher {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubscribeForEvent", arg0, arg1)
	ret0, _ := ret[0].(dispatcher.Dispatcher)
	return ret0
}

// SubscribeForEvent indicates an expected call of SubscribeForEvent.
func (mr *MockDispatcherMockRecorder) SubscribeForEvent(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubscribeForEvent", reflect.TypeOf((*MockDispatcher)(nil).SubscribeForEvent), arg0, arg1)
}