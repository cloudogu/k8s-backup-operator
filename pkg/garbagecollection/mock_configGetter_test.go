// Code generated by mockery v2.20.0. DO NOT EDIT.

package garbagecollection

import (
	context "context"

	retention "github.com/cloudogu/k8s-backup-operator/pkg/retention"
	mock "github.com/stretchr/testify/mock"
)

// mockConfigGetter is an autogenerated mock type for the configGetter type
type mockConfigGetter struct {
	mock.Mock
}

type mockConfigGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *mockConfigGetter) EXPECT() *mockConfigGetter_Expecter {
	return &mockConfigGetter_Expecter{mock: &_m.Mock}
}

// GetConfig provides a mock function with given fields: ctx
func (_m *mockConfigGetter) GetConfig(ctx context.Context) (retention.Config, error) {
	ret := _m.Called(ctx)

	var r0 retention.Config
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (retention.Config, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) retention.Config); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(retention.Config)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockConfigGetter_GetConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetConfig'
type mockConfigGetter_GetConfig_Call struct {
	*mock.Call
}

// GetConfig is a helper method to define mock.On call
//   - ctx context.Context
func (_e *mockConfigGetter_Expecter) GetConfig(ctx interface{}) *mockConfigGetter_GetConfig_Call {
	return &mockConfigGetter_GetConfig_Call{Call: _e.mock.On("GetConfig", ctx)}
}

func (_c *mockConfigGetter_GetConfig_Call) Run(run func(ctx context.Context)) *mockConfigGetter_GetConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *mockConfigGetter_GetConfig_Call) Return(_a0 retention.Config, _a1 error) *mockConfigGetter_GetConfig_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockConfigGetter_GetConfig_Call) RunAndReturn(run func(context.Context) (retention.Config, error)) *mockConfigGetter_GetConfig_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockConfigGetter interface {
	mock.TestingT
	Cleanup(func())
}

// newMockConfigGetter creates a new instance of mockConfigGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockConfigGetter(t mockConstructorTestingTnewMockConfigGetter) *mockConfigGetter {
	mock := &mockConfigGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
