// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"
	io "io"

	core "github.com/LambdaTest/test-at-scale/pkg/core"

	mock "github.com/stretchr/testify/mock"
)

// AzureClient is an autogenerated mock type for the AzureClient type
type AzureClient struct {
	mock.Mock
}

// Create provides a mock function with given fields: ctx, path, reader, mimeType
func (_m *AzureClient) Create(ctx context.Context, path string, reader io.Reader, mimeType string) (string, error) {
	ret := _m.Called(ctx, path, reader, mimeType)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, io.Reader, string) string); ok {
		r0 = rf(ctx, path, reader, mimeType)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, io.Reader, string) error); ok {
		r1 = rf(ctx, path, reader, mimeType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateUsingSASURL provides a mock function with given fields: ctx, sasURL, reader, mimeType
func (_m *AzureClient) CreateUsingSASURL(ctx context.Context, sasURL string, reader io.Reader, mimeType string) (string, error) {
	ret := _m.Called(ctx, sasURL, reader, mimeType)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, string, io.Reader, string) string); ok {
		r0 = rf(ctx, sasURL, reader, mimeType)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, io.Reader, string) error); ok {
		r1 = rf(ctx, sasURL, reader, mimeType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Exists provides a mock function with given fields: ctx, path
func (_m *AzureClient) Exists(ctx context.Context, path string) (bool, error) {
	ret := _m.Called(ctx, path)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, path)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Find provides a mock function with given fields: ctx, path
func (_m *AzureClient) Find(ctx context.Context, path string) (io.ReadCloser, error) {
	ret := _m.Called(ctx, path)

	var r0 io.ReadCloser
	if rf, ok := ret.Get(0).(func(context.Context, string) io.ReadCloser); ok {
		r0 = rf(ctx, path)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindUsingSASUrl provides a mock function with given fields: ctx, sasURL
func (_m *AzureClient) FindUsingSASUrl(ctx context.Context, sasURL string) (io.ReadCloser, error) {
	ret := _m.Called(ctx, sasURL)

	var r0 io.ReadCloser
	if rf, ok := ret.Get(0).(func(context.Context, string) io.ReadCloser); ok {
		r0 = rf(ctx, sasURL)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, sasURL)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSASURL provides a mock function with given fields: ctx, purpose, query
func (_m *AzureClient) GetSASURL(ctx context.Context, purpose core.SASURLPurpose, query map[string]interface{}) (string, error) {
	ret := _m.Called(ctx, purpose, query)

	var r0 string
	if rf, ok := ret.Get(0).(func(context.Context, core.SASURLPurpose, map[string]interface{}) string); ok {
		r0 = rf(ctx, purpose, query)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, core.SASURLPurpose, map[string]interface{}) error); ok {
		r1 = rf(ctx, purpose, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewAzureClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewAzureClient creates a new instance of AzureClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAzureClient(t mockConstructorTestingTNewAzureClient) *AzureClient {
	mock := &AzureClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
