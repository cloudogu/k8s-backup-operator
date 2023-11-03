// Code generated by mockery v2.20.0. DO NOT EDIT.

package backupschedule

import (
	context "context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// mockUpdateManager is an autogenerated mock type for the updateManager type
type mockUpdateManager struct {
	mock.Mock
}

type mockUpdateManager_Expecter struct {
	mock *mock.Mock
}

func (_m *mockUpdateManager) EXPECT() *mockUpdateManager_Expecter {
	return &mockUpdateManager_Expecter{mock: &_m.Mock}
}

// update provides a mock function with given fields: ctx, backupSchedule
func (_m *mockUpdateManager) update(ctx context.Context, backupSchedule *v1.BackupSchedule) error {
	ret := _m.Called(ctx, backupSchedule)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.BackupSchedule) error); ok {
		r0 = rf(ctx, backupSchedule)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockUpdateManager_update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'update'
type mockUpdateManager_update_Call struct {
	*mock.Call
}

// update is a helper method to define mock.On call
//   - ctx context.Context
//   - backupSchedule *v1.BackupSchedule
func (_e *mockUpdateManager_Expecter) update(ctx interface{}, backupSchedule interface{}) *mockUpdateManager_update_Call {
	return &mockUpdateManager_update_Call{Call: _e.mock.On("update", ctx, backupSchedule)}
}

func (_c *mockUpdateManager_update_Call) Run(run func(ctx context.Context, backupSchedule *v1.BackupSchedule)) *mockUpdateManager_update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.BackupSchedule))
	})
	return _c
}

func (_c *mockUpdateManager_update_Call) Return(_a0 error) *mockUpdateManager_update_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockUpdateManager_update_Call) RunAndReturn(run func(context.Context, *v1.BackupSchedule) error) *mockUpdateManager_update_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockUpdateManager interface {
	mock.TestingT
	Cleanup(func())
}

// newMockUpdateManager creates a new instance of mockUpdateManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockUpdateManager(t mockConstructorTestingTnewMockUpdateManager) *mockUpdateManager {
	mock := &mockUpdateManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
