// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	sync "sync"
)

// SynapseManager is an autogenerated mock type for the SynapseManager type
type SynapseManager struct {
	mock.Mock
}

// InitiateConnection provides a mock function with given fields: ctx, wg, connectionFailed
func (_m *SynapseManager) InitiateConnection(ctx context.Context, wg *sync.WaitGroup, connectionFailed chan struct{}) {
	_m.Called(ctx, wg, connectionFailed)
}

type mockConstructorTestingTNewSynapseManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewSynapseManager creates a new instance of SynapseManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSynapseManager(t mockConstructorTestingTNewSynapseManager) *SynapseManager {
	mock := &SynapseManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
