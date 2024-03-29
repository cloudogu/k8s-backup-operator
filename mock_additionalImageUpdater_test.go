// Code generated by mockery v2.20.0. DO NOT EDIT.

package main

import (
	context "context"

	additionalimages "github.com/cloudogu/k8s-backup-operator/pkg/additionalimages"

	mock "github.com/stretchr/testify/mock"
)

// mockAdditionalImageUpdater is an autogenerated mock type for the additionalImageUpdater type
type mockAdditionalImageUpdater struct {
	mock.Mock
}

type mockAdditionalImageUpdater_Expecter struct {
	mock *mock.Mock
}

func (_m *mockAdditionalImageUpdater) EXPECT() *mockAdditionalImageUpdater_Expecter {
	return &mockAdditionalImageUpdater_Expecter{mock: &_m.Mock}
}

// Update provides a mock function with given fields: ctx, config
func (_m *mockAdditionalImageUpdater) Update(ctx context.Context, config additionalimages.ImageConfig) error {
	ret := _m.Called(ctx, config)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, additionalimages.ImageConfig) error); ok {
		r0 = rf(ctx, config)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockAdditionalImageUpdater_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockAdditionalImageUpdater_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - config additionalimages.ImageConfig
func (_e *mockAdditionalImageUpdater_Expecter) Update(ctx interface{}, config interface{}) *mockAdditionalImageUpdater_Update_Call {
	return &mockAdditionalImageUpdater_Update_Call{Call: _e.mock.On("Update", ctx, config)}
}

func (_c *mockAdditionalImageUpdater_Update_Call) Run(run func(ctx context.Context, config additionalimages.ImageConfig)) *mockAdditionalImageUpdater_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(additionalimages.ImageConfig))
	})
	return _c
}

func (_c *mockAdditionalImageUpdater_Update_Call) Return(_a0 error) *mockAdditionalImageUpdater_Update_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockAdditionalImageUpdater_Update_Call) RunAndReturn(run func(context.Context, additionalimages.ImageConfig) error) *mockAdditionalImageUpdater_Update_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockAdditionalImageUpdater interface {
	mock.TestingT
	Cleanup(func())
}

// newMockAdditionalImageUpdater creates a new instance of mockAdditionalImageUpdater. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockAdditionalImageUpdater(t mockConstructorTestingTnewMockAdditionalImageUpdater) *mockAdditionalImageUpdater {
	mock := &mockAdditionalImageUpdater{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
