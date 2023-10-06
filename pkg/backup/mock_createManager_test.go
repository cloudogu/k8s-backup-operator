// Code generated by mockery v2.20.0. DO NOT EDIT.

package backup

import (
	context "context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// mockCreateManager is an autogenerated mock type for the createManager type
type mockCreateManager struct {
	mock.Mock
}

type mockCreateManager_Expecter struct {
	mock *mock.Mock
}

func (_m *mockCreateManager) EXPECT() *mockCreateManager_Expecter {
	return &mockCreateManager_Expecter{mock: &_m.Mock}
}

// create provides a mock function with given fields: ctx, backup
func (_m *mockCreateManager) create(ctx context.Context, backup *v1.Backup) error {
	ret := _m.Called(ctx, backup)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) error); ok {
		r0 = rf(ctx, backup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockCreateManager_create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'create'
type mockCreateManager_create_Call struct {
	*mock.Call
}

// create is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockCreateManager_Expecter) create(ctx interface{}, backup interface{}) *mockCreateManager_create_Call {
	return &mockCreateManager_create_Call{Call: _e.mock.On("create", ctx, backup)}
}

func (_c *mockCreateManager_create_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockCreateManager_create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockCreateManager_create_Call) Return(_a0 error) *mockCreateManager_create_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockCreateManager_create_Call) RunAndReturn(run func(context.Context, *v1.Backup) error) *mockCreateManager_create_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockCreateManager interface {
	mock.TestingT
	Cleanup(func())
}

// newMockCreateManager creates a new instance of mockCreateManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockCreateManager(t mockConstructorTestingTnewMockCreateManager) *mockCreateManager {
	mock := &mockCreateManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}