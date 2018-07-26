// Code generated by MockGen. DO NOT EDIT.
// Source: server.go

// Package goconnpool is a generated GoMock package.
package goconnpool

import (
	context "context"
	net "net"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// Mockdialer is a mock of dialer interface
type Mockdialer struct {
	ctrl     *gomock.Controller
	recorder *MockdialerMockRecorder
}

// MockdialerMockRecorder is the mock recorder for Mockdialer
type MockdialerMockRecorder struct {
	mock *Mockdialer
}

// NewMockdialer creates a new mock instance
func NewMockdialer(ctrl *gomock.Controller) *Mockdialer {
	mock := &Mockdialer{ctrl: ctrl}
	mock.recorder = &MockdialerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *Mockdialer) EXPECT() *MockdialerMockRecorder {
	return m.recorder
}

// DialContext mocks base method
func (m *Mockdialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	ret := m.ctrl.Call(m, "DialContext", ctx, network, address)
	ret0, _ := ret[0].(net.Conn)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DialContext indicates an expected call of DialContext
func (mr *MockdialerMockRecorder) DialContext(ctx, network, address interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DialContext", reflect.TypeOf((*Mockdialer)(nil).DialContext), ctx, network, address)
}

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
