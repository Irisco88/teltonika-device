// Code generated by MockGen. DO NOT EDIT.
// Source: conn.go

// Package mock_clickhouse is a generated GoMock package.
package mock_clickhouse

import (
	context "context"
	reflect "reflect"

	driver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	gomock "github.com/golang/mock/gomock"
	devicev1 "github.com/irisco88/protos/gen/device/v1"
)

// MockAVLDBConn is a mock of AVLDBConn interface.
type MockAVLDBConn struct {
	ctrl     *gomock.Controller
	recorder *MockAVLDBConnMockRecorder
}

// MockAVLDBConnMockRecorder is the mock recorder for MockAVLDBConn.
type MockAVLDBConnMockRecorder struct {
	mock *MockAVLDBConn
}

// NewMockAVLDBConn creates a new mock instance.
func NewMockAVLDBConn(ctrl *gomock.Controller) *MockAVLDBConn {
	mock := &MockAVLDBConn{ctrl: ctrl}
	mock.recorder = &MockAVLDBConnMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAVLDBConn) EXPECT() *MockAVLDBConnMockRecorder {
	return m.recorder
}

// GetConn mocks base method.
func (m *MockAVLDBConn) GetConn() driver.Conn {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConn")
	ret0, _ := ret[0].(driver.Conn)
	return ret0
}

// GetConn indicates an expected call of GetConn.
func (mr *MockAVLDBConnMockRecorder) GetConn() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConn", reflect.TypeOf((*MockAVLDBConn)(nil).GetConn))
}

// SaveAvlPoints mocks base method.
func (m *MockAVLDBConn) SaveAvlPoints(ctx context.Context, points []*devicev1.AVLData) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveAvlPoints", ctx, points)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveAvlPoints indicates an expected call of SaveAvlPoints.
func (mr *MockAVLDBConnMockRecorder) SaveAvlPoints(ctx, points interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveAvlPoints", reflect.TypeOf((*MockAVLDBConn)(nil).SaveAvlPoints), ctx, points)
}

// SaveRawData mocks base method.
func (m *MockAVLDBConn) SaveRawData(ctx context.Context, imei, payload string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveRawData", ctx, imei, payload)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveRawData indicates an expected call of SaveRawData.
func (mr *MockAVLDBConnMockRecorder) SaveRawData(ctx, imei, payload interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveRawData", reflect.TypeOf((*MockAVLDBConn)(nil).SaveRawData), ctx, imei, payload)
}
