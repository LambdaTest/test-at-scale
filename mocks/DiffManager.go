// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/LambdaTest/test-at-scale/pkg/core"
	mock "github.com/stretchr/testify/mock"
)

// DiffManager is an autogenerated mock type for the DiffManager type
type DiffManager struct {
	mock.Mock
}

// GetChangedFiles provides a mock function with given fields: ctx, payload, oauth
func (_m *DiffManager) GetChangedFiles(ctx context.Context, payload *core.Payload, oauth *core.Oauth) (map[string]int, error) {
	ret := _m.Called(ctx, payload, oauth)

	var r0 map[string]int
	if rf, ok := ret.Get(0).(func(context.Context, *core.Payload, *core.Oauth) map[string]int); ok {
		r0 = rf(ctx, payload, oauth)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]int)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *core.Payload, *core.Oauth) error); ok {
		r1 = rf(ctx, payload, oauth)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewDiffManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewDiffManager creates a new instance of DiffManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDiffManager(t mockConstructorTestingTNewDiffManager) *DiffManager {
	mock := &DiffManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
