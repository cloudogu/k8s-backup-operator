// Code generated by mockery v2.53.3. DO NOT EDIT.

package restore

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

// delete provides a mock function with given fields: ctx, restore
func (_m *mockDeleteManager) delete(ctx context.Context, restore *v1.Restore) error {
	ret := _m.Called(ctx, restore)

	if len(ret) == 0 {
		panic("no return value specified for delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Restore) error); ok {
		r0 = rf(ctx, restore)
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
//   - restore *v1.Restore
func (_e *mockDeleteManager_Expecter) delete(ctx interface{}, restore interface{}) *mockDeleteManager_delete_Call {
	return &mockDeleteManager_delete_Call{Call: _e.mock.On("delete", ctx, restore)}
}

func (_c *mockDeleteManager_delete_Call) Run(run func(ctx context.Context, restore *v1.Restore)) *mockDeleteManager_delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Restore))
	})
	return _c
}

func (_c *mockDeleteManager_delete_Call) Return(_a0 error) *mockDeleteManager_delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockDeleteManager_delete_Call) RunAndReturn(run func(context.Context, *v1.Restore) error) *mockDeleteManager_delete_Call {
	_c.Call.Return(run)
	return _c
}

// newMockDeleteManager creates a new instance of mockDeleteManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockDeleteManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockDeleteManager {
	mock := &mockDeleteManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
