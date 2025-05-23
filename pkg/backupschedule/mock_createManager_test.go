// Code generated by mockery v2.53.3. DO NOT EDIT.

package backupschedule

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

// create provides a mock function with given fields: ctx, backupSchedule
func (_m *mockCreateManager) create(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	ret := _m.Called(ctx, backupSchedule)

	if len(ret) == 0 {
		panic("no return value specified for create")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.BackupSchedule) error); ok {
		r0 = rf(ctx, backupSchedule)
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
//   - backupSchedule *v1.BackupSchedule
func (_e *mockCreateManager_Expecter) create(ctx interface{}, backupSchedule interface{}) *mockCreateManager_create_Call {
	return &mockCreateManager_create_Call{Call: _e.mock.On("create", ctx, backupSchedule)}
}

func (_c *mockCreateManager_create_Call) Run(run func(ctx context.Context, backupSchedule *v1.BackupSchedule)) *mockCreateManager_create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.BackupSchedule))
	})
	return _c
}

func (_c *mockCreateManager_create_Call) Return(_a0 error) *mockCreateManager_create_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockCreateManager_create_Call) RunAndReturn(run func(context.Context, *v1.BackupSchedule) error) *mockCreateManager_create_Call {
	_c.Call.Return(run)
	return _c
}

// newMockCreateManager creates a new instance of mockCreateManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockCreateManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockCreateManager {
	mock := &mockCreateManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
