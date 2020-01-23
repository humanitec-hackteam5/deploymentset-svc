// Code generated by MockGen. DO NOT EDIT.
// Source: main.go

// Package main is a generated GoMock package.
package main

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	depset "humanitec.io/deploymentset-svc/pkg/depset"
)

// Mockmodeler is a mock of modeler interface
type Mockmodeler struct {
	ctrl     *gomock.Controller
	recorder *MockmodelerMockRecorder
}

// MockmodelerMockRecorder is the mock recorder for Mockmodeler
type MockmodelerMockRecorder struct {
	mock *Mockmodeler
}

// NewMockmodeler creates a new mock instance
func NewMockmodeler(ctrl *gomock.Controller) *Mockmodeler {
	mock := &Mockmodeler{ctrl: ctrl}
	mock.recorder = &MockmodelerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *Mockmodeler) EXPECT() *MockmodelerMockRecorder {
	return m.recorder
}

// insertSet mocks base method
func (m *Mockmodeler) insertSet(orgID, appID string, sw SetWrapper) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "insertSet", orgID, appID, sw)
	ret0, _ := ret[0].(error)
	return ret0
}

// insertSet indicates an expected call of insertSet
func (mr *MockmodelerMockRecorder) insertSet(orgID, appID, sw interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "insertSet", reflect.TypeOf((*Mockmodeler)(nil).insertSet), orgID, appID, sw)
}

// selectAllSets mocks base method
func (m *Mockmodeler) selectAllSets(orgID, appID string) ([]SetWrapper, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "selectAllSets", orgID, appID)
	ret0, _ := ret[0].([]SetWrapper)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// selectAllSets indicates an expected call of selectAllSets
func (mr *MockmodelerMockRecorder) selectAllSets(orgID, appID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "selectAllSets", reflect.TypeOf((*Mockmodeler)(nil).selectAllSets), orgID, appID)
}

// selectSet mocks base method
func (m *Mockmodeler) selectSet(orgID, appID, setID string) (SetWrapper, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "selectSet", orgID, appID, setID)
	ret0, _ := ret[0].(SetWrapper)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// selectSet indicates an expected call of selectSet
func (mr *MockmodelerMockRecorder) selectSet(orgID, appID, setID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "selectSet", reflect.TypeOf((*Mockmodeler)(nil).selectSet), orgID, appID, setID)
}

// selectRawSet mocks base method
func (m *Mockmodeler) selectRawSet(orgID, appID, setID string) (depset.Set, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "selectRawSet", orgID, appID, setID)
	ret0, _ := ret[0].(depset.Set)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// selectRawSet indicates an expected call of selectRawSet
func (mr *MockmodelerMockRecorder) selectRawSet(orgID, appID, setID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "selectRawSet", reflect.TypeOf((*Mockmodeler)(nil).selectRawSet), orgID, appID, setID)
}