// Code generated by MockGen. DO NOT EDIT.
// Source: server.go

// Package goconnpool is a generated GoMock package.
package goconnpool

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
)

// MockconnectionProvider is a mock of connectionProvider interface
type MockconnectionProvider struct {
	ctrl     *gomock.Controller
	recorder *MockconnectionProviderMockRecorder
}

// MockconnectionProviderMockRecorder is the mock recorder for MockconnectionProvider
type MockconnectionProviderMockRecorder struct {
	mock *MockconnectionProvider
}

// NewMockconnectionProvider creates a new mock instance
func NewMockconnectionProvider(ctrl *gomock.Controller) *MockconnectionProvider {
	mock := &MockconnectionProvider{ctrl: ctrl}
	mock.recorder = &MockconnectionProviderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockconnectionProvider) EXPECT() *MockconnectionProviderMockRecorder {
	return m.recorder
}

// getConnection mocks base method
func (m *MockconnectionProvider) getConnection(ctx context.Context) (Conn, error) {
	ret := m.ctrl.Call(m, "getConnection", ctx)
	ret0, _ := ret[0].(Conn)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// getConnection indicates an expected call of getConnection
func (mr *MockconnectionProviderMockRecorder) getConnection(ctx interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getConnection", reflect.TypeOf((*MockconnectionProvider)(nil).getConnection), ctx)
}

// retryTimeout mocks base method
func (m *MockconnectionProvider) retryTimeout() time.Duration {
	ret := m.ctrl.Call(m, "retryTimeout")
	ret0, _ := ret[0].(time.Duration)
	return ret0
}

// retryTimeout indicates an expected call of retryTimeout
func (mr *MockconnectionProviderMockRecorder) retryTimeout() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "retryTimeout", reflect.TypeOf((*MockconnectionProvider)(nil).retryTimeout))
}
