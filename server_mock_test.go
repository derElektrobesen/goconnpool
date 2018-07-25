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