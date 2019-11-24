// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/NiR-/webdf/pkg/filefetch (interfaces: FileFetcher)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockFileFetcher is a mock of FileFetcher interface
type MockFileFetcher struct {
	ctrl     *gomock.Controller
	recorder *MockFileFetcherMockRecorder
}

// MockFileFetcherMockRecorder is the mock recorder for MockFileFetcher
type MockFileFetcherMockRecorder struct {
	mock *MockFileFetcher
}

// NewMockFileFetcher creates a new mock instance
func NewMockFileFetcher(ctrl *gomock.Controller) *MockFileFetcher {
	mock := &MockFileFetcher{ctrl: ctrl}
	mock.recorder = &MockFileFetcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFileFetcher) EXPECT() *MockFileFetcherMockRecorder {
	return m.recorder
}

// FetchFile mocks base method
func (m *MockFileFetcher) FetchFile(arg0 context.Context, arg1, arg2 string) ([]byte, error) {
	ret := m.ctrl.Call(m, "FetchFile", arg0, arg1, arg2)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FetchFile indicates an expected call of FetchFile
func (mr *MockFileFetcherMockRecorder) FetchFile(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FetchFile", reflect.TypeOf((*MockFileFetcher)(nil).FetchFile), arg0, arg1, arg2)
}
