// Code generated by mockery v2.20.0. DO NOT EDIT.

package backup

import (
	context "context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// mockDeleteManager is an autogenerated mock type for the deleteManager type
type mockDeleteManager struct {
	mock.Mock
}

type mockDeleteManager_Expecter struct {
	mock *mock.Mock
}

func (_m *mockDeleteManager) EXPECT() *mockDeleteManager_Expecter {
	return &mockDeleteManager_Expecter{mock: &_m.Mock}
}

// delete provides a mock function with given fields: ctx, backup
func (_m *mockDeleteManager) delete(ctx context.Context, backup *v1.Backup) error {
	ret := _m.Called(ctx, backup)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) error); ok {
		r0 = rf(ctx, backup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockDeleteManager_delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'delete'
type mockDeleteManager_delete_Call struct {
	*mock.Call
}

// delete is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockDeleteManager_Expecter) delete(ctx interface{}, backup interface{}) *mockDeleteManager_delete_Call {
	return &mockDeleteManager_delete_Call{Call: _e.mock.On("delete", ctx, backup)}
}

func (_c *mockDeleteManager_delete_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockDeleteManager_delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockDeleteManager_delete_Call) Return(_a0 error) *mockDeleteManager_delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockDeleteManager_delete_Call) RunAndReturn(run func(context.Context, *v1.Backup) error) *mockDeleteManager_delete_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockDeleteManager interface {
	mock.TestingT
	Cleanup(func())
}

// newMockDeleteManager creates a new instance of mockDeleteManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockDeleteManager(t mockConstructorTestingTnewMockDeleteManager) *mockDeleteManager {
	mock := &mockDeleteManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
