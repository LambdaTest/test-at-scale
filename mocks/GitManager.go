// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/LambdaTest/test-at-scale/pkg/core"
	mock "github.com/stretchr/testify/mock"
)

// GitManager is an autogenerated mock type for the GitManager type
type GitManager struct {
	mock.Mock
}

// Clone provides a mock function with given fields: ctx, payload, oauth
func (_m *GitManager) Clone(ctx context.Context, payload *core.Payload, oauth *core.Oauth) error {
	ret := _m.Called(ctx, payload, oauth)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.Payload, *core.Oauth) error); ok {
		r0 = rf(ctx, payload, oauth)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewGitManager interface {
	mock.TestingT
	Cleanup(func())
}

// NewGitManager creates a new instance of GitManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewGitManager(t mockConstructorTestingTNewGitManager) *GitManager {
	mock := &GitManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
