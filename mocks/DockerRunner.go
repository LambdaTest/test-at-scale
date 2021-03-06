// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	core "github.com/LambdaTest/test-at-scale/pkg/core"
	mock "github.com/stretchr/testify/mock"
)

// DockerRunner is an autogenerated mock type for the DockerRunner type
type DockerRunner struct {
	mock.Mock
}

// Create provides a mock function with given fields: _a0, _a1
func (_m *DockerRunner) Create(_a0 context.Context, _a1 *core.RunnerOptions) core.ContainerStatus {
	ret := _m.Called(_a0, _a1)

	var r0 core.ContainerStatus
	if rf, ok := ret.Get(0).(func(context.Context, *core.RunnerOptions) core.ContainerStatus); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(core.ContainerStatus)
	}

	return r0
}

// Destroy provides a mock function with given fields: ctx, r
func (_m *DockerRunner) Destroy(ctx context.Context, r *core.RunnerOptions) error {
	ret := _m.Called(ctx, r)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.RunnerOptions) error); ok {
		r0 = rf(ctx, r)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetInfo provides a mock function with given fields: _a0
func (_m *DockerRunner) GetInfo(_a0 context.Context) (float32, int64) {
	ret := _m.Called(_a0)

	var r0 float32
	if rf, ok := ret.Get(0).(func(context.Context) float32); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(float32)
	}

	var r1 int64
	if rf, ok := ret.Get(1).(func(context.Context) int64); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Get(1).(int64)
	}

	return r0, r1
}

// Initiate provides a mock function with given fields: _a0, _a1, _a2
func (_m *DockerRunner) Initiate(_a0 context.Context, _a1 *core.RunnerOptions, _a2 chan core.ContainerStatus) {
	_m.Called(_a0, _a1, _a2)
}

// KillRunningDocker provides a mock function with given fields: ctx
func (_m *DockerRunner) KillRunningDocker(ctx context.Context) {
	_m.Called(ctx)
}

// PullImage provides a mock function with given fields: containerImageConfig, r
func (_m *DockerRunner) PullImage(containerImageConfig *core.ContainerImageConfig, r *core.RunnerOptions) error {
	ret := _m.Called(containerImageConfig, r)

	var r0 error
	if rf, ok := ret.Get(0).(func(*core.ContainerImageConfig, *core.RunnerOptions) error); ok {
		r0 = rf(containerImageConfig, r)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Run provides a mock function with given fields: _a0, _a1
func (_m *DockerRunner) Run(_a0 context.Context, _a1 *core.RunnerOptions) core.ContainerStatus {
	ret := _m.Called(_a0, _a1)

	var r0 core.ContainerStatus
	if rf, ok := ret.Get(0).(func(context.Context, *core.RunnerOptions) core.ContainerStatus); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(core.ContainerStatus)
	}

	return r0
}

// WaitForCompletion provides a mock function with given fields: ctx, r
func (_m *DockerRunner) WaitForCompletion(ctx context.Context, r *core.RunnerOptions) error {
	ret := _m.Called(ctx, r)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *core.RunnerOptions) error); ok {
		r0 = rf(ctx, r)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewDockerRunner interface {
	mock.TestingT
	Cleanup(func())
}

// NewDockerRunner creates a new instance of DockerRunner. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDockerRunner(t mockConstructorTestingTNewDockerRunner) *DockerRunner {
	mock := &DockerRunner{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
