// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/go-foreman/foreman/saga/mutex (interfaces: Mutex,Lock)

// Package mutex is a generated GoMock package.
package mutex

import (
	context "context"
	reflect "reflect"

	mutex "github.com/go-foreman/foreman/saga/mutex"
	gomock "github.com/golang/mock/gomock"
)

// MockMutex is a mock of Mutex interface.
type MockMutex struct {
	ctrl     *gomock.Controller
	recorder *MockMutexMockRecorder
}

// MockMutexMockRecorder is the mock recorder for MockMutex.
type MockMutexMockRecorder struct {
	mock *MockMutex
}

// NewMockMutex creates a new mock instance.
func NewMockMutex(ctrl *gomock.Controller) *MockMutex {
	mock := &MockMutex{ctrl: ctrl}
	mock.recorder = &MockMutexMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMutex) EXPECT() *MockMutexMockRecorder {
	return m.recorder
}

// Lock mocks base method.
func (m *MockMutex) Lock(arg0 context.Context, arg1 string) (mutex.Lock, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Lock", arg0, arg1)
	ret0, _ := ret[0].(mutex.Lock)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Lock indicates an expected call of Lock.
func (mr *MockMutexMockRecorder) Lock(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Lock", reflect.TypeOf((*MockMutex)(nil).Lock), arg0, arg1)
}

// MockLock is a mock of Lock interface.
type MockLock struct {
	ctrl     *gomock.Controller
	recorder *MockLockMockRecorder
}

// MockLockMockRecorder is the mock recorder for MockLock.
type MockLockMockRecorder struct {
	mock *MockLock
}

// NewMockLock creates a new mock instance.
func NewMockLock(ctrl *gomock.Controller) *MockLock {
	mock := &MockLock{ctrl: ctrl}
	mock.recorder = &MockLockMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLock) EXPECT() *MockLockMockRecorder {
	return m.recorder
}

// Release mocks base method.
func (m *MockLock) Release(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Release", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Release indicates an expected call of Release.
func (mr *MockLockMockRecorder) Release(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Release", reflect.TypeOf((*MockLock)(nil).Release), arg0)
}
