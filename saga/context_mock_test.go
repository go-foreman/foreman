// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/go-foreman/foreman/saga (interfaces: SagaContext)

// Package saga is a generated GoMock package.
package saga

import (
	context "context"
	reflect "reflect"

	log "github.com/go-foreman/foreman/log"
	endpoint "github.com/go-foreman/foreman/pubsub/endpoint"
	message "github.com/go-foreman/foreman/pubsub/message"
	gomock "github.com/golang/mock/gomock"
)

// MockSagaContext is a mock of SagaContext interface.
type MockSagaContext struct {
	ctrl     *gomock.Controller
	recorder *MockSagaContextMockRecorder
}

// MockSagaContextMockRecorder is the mock recorder for MockSagaContext.
type MockSagaContextMockRecorder struct {
	mock *MockSagaContext
}

// NewMockSagaContext creates a new mock instance.
func NewMockSagaContext(ctrl *gomock.Controller) *MockSagaContext {
	mock := &MockSagaContext{ctrl: ctrl}
	mock.recorder = &MockSagaContextMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSagaContext) EXPECT() *MockSagaContextMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockSagaContext) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockSagaContextMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockSagaContext)(nil).Context))
}

// Deliveries mocks base method.
func (m *MockSagaContext) Deliveries() []*Delivery {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Deliveries")
	ret0, _ := ret[0].([]*Delivery)
	return ret0
}

// Deliveries indicates an expected call of Deliveries.
func (mr *MockSagaContextMockRecorder) Deliveries() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Deliveries", reflect.TypeOf((*MockSagaContext)(nil).Deliveries))
}

// Dispatch mocks base method.
func (m *MockSagaContext) Dispatch(arg0 message.Object, arg1 ...endpoint.DeliveryOption) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	m.ctrl.Call(m, "Dispatch", varargs...)
}

// Dispatch indicates an expected call of Dispatch.
func (mr *MockSagaContextMockRecorder) Dispatch(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Dispatch", reflect.TypeOf((*MockSagaContext)(nil).Dispatch), varargs...)
}

// Logger mocks base method.
func (m *MockSagaContext) Logger() log.Logger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Logger")
	ret0, _ := ret[0].(log.Logger)
	return ret0
}

// Logger indicates an expected call of Logger.
func (mr *MockSagaContextMockRecorder) Logger() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Logger", reflect.TypeOf((*MockSagaContext)(nil).Logger))
}

// Message mocks base method.
func (m *MockSagaContext) Message() *message.ReceivedMessage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Message")
	ret0, _ := ret[0].(*message.ReceivedMessage)
	return ret0
}

// Message indicates an expected call of Message.
func (mr *MockSagaContextMockRecorder) Message() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Message", reflect.TypeOf((*MockSagaContext)(nil).Message))
}

// Return mocks base method.
func (m *MockSagaContext) Return(arg0 ...endpoint.DeliveryOption) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Return", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Return indicates an expected call of Return.
func (mr *MockSagaContextMockRecorder) Return(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Return", reflect.TypeOf((*MockSagaContext)(nil).Return), arg0...)
}

// SagaInstance mocks base method.
func (m *MockSagaContext) SagaInstance() Instance {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SagaInstance")
	ret0, _ := ret[0].(Instance)
	return ret0
}

// SagaInstance indicates an expected call of SagaInstance.
func (mr *MockSagaContextMockRecorder) SagaInstance() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SagaInstance", reflect.TypeOf((*MockSagaContext)(nil).SagaInstance))
}

// Valid mocks base method.
func (m *MockSagaContext) Valid() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Valid")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Valid indicates an expected call of Valid.
func (mr *MockSagaContextMockRecorder) Valid() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Valid", reflect.TypeOf((*MockSagaContext)(nil).Valid))
}
