// Code generated by mockery v2.20.0. DO NOT EDIT.

package restore

import (
	context "context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// mockRestoreProvider is an autogenerated mock type for the restoreProvider type
type mockRestoreProvider struct {
	mock.Mock
}

type mockRestoreProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *mockRestoreProvider) EXPECT() *mockRestoreProvider_Expecter {
	return &mockRestoreProvider_Expecter{mock: &_m.Mock}
}

// CheckReady provides a mock function with given fields: ctx
func (_m *mockRestoreProvider) CheckReady(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRestoreProvider_CheckReady_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckReady'
type mockRestoreProvider_CheckReady_Call struct {
	*mock.Call
}

// CheckReady is a helper method to define mock.On call
//   - ctx context.Context
func (_e *mockRestoreProvider_Expecter) CheckReady(ctx interface{}) *mockRestoreProvider_CheckReady_Call {
	return &mockRestoreProvider_CheckReady_Call{Call: _e.mock.On("CheckReady", ctx)}
}

func (_c *mockRestoreProvider_CheckReady_Call) Run(run func(ctx context.Context)) *mockRestoreProvider_CheckReady_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *mockRestoreProvider_CheckReady_Call) Return(_a0 error) *mockRestoreProvider_CheckReady_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRestoreProvider_CheckReady_Call) RunAndReturn(run func(context.Context) error) *mockRestoreProvider_CheckReady_Call {
	_c.Call.Return(run)
	return _c
}

// CreateBackup provides a mock function with given fields: ctx, backup
func (_m *mockRestoreProvider) CreateBackup(ctx context.Context, backup *v1.Backup) error {
	ret := _m.Called(ctx, backup)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) error); ok {
		r0 = rf(ctx, backup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRestoreProvider_CreateBackup_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateBackup'
type mockRestoreProvider_CreateBackup_Call struct {
	*mock.Call
}

// CreateBackup is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockRestoreProvider_Expecter) CreateBackup(ctx interface{}, backup interface{}) *mockRestoreProvider_CreateBackup_Call {
	return &mockRestoreProvider_CreateBackup_Call{Call: _e.mock.On("CreateBackup", ctx, backup)}
}

func (_c *mockRestoreProvider_CreateBackup_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockRestoreProvider_CreateBackup_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockRestoreProvider_CreateBackup_Call) Return(_a0 error) *mockRestoreProvider_CreateBackup_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRestoreProvider_CreateBackup_Call) RunAndReturn(run func(context.Context, *v1.Backup) error) *mockRestoreProvider_CreateBackup_Call {
	_c.Call.Return(run)
	return _c
}

// CreateRestore provides a mock function with given fields: ctx, restore
func (_m *mockRestoreProvider) CreateRestore(ctx context.Context, restore *v1.Restore) error {
	ret := _m.Called(ctx, restore)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Restore) error); ok {
		r0 = rf(ctx, restore)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRestoreProvider_CreateRestore_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateRestore'
type mockRestoreProvider_CreateRestore_Call struct {
	*mock.Call
}

// CreateRestore is a helper method to define mock.On call
//   - ctx context.Context
//   - restore *v1.Restore
func (_e *mockRestoreProvider_Expecter) CreateRestore(ctx interface{}, restore interface{}) *mockRestoreProvider_CreateRestore_Call {
	return &mockRestoreProvider_CreateRestore_Call{Call: _e.mock.On("CreateRestore", ctx, restore)}
}

func (_c *mockRestoreProvider_CreateRestore_Call) Run(run func(ctx context.Context, restore *v1.Restore)) *mockRestoreProvider_CreateRestore_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Restore))
	})
	return _c
}

func (_c *mockRestoreProvider_CreateRestore_Call) Return(_a0 error) *mockRestoreProvider_CreateRestore_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRestoreProvider_CreateRestore_Call) RunAndReturn(run func(context.Context, *v1.Restore) error) *mockRestoreProvider_CreateRestore_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteBackup provides a mock function with given fields: ctx, backup
func (_m *mockRestoreProvider) DeleteBackup(ctx context.Context, backup *v1.Backup) error {
	ret := _m.Called(ctx, backup)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) error); ok {
		r0 = rf(ctx, backup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRestoreProvider_DeleteBackup_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteBackup'
type mockRestoreProvider_DeleteBackup_Call struct {
	*mock.Call
}

// DeleteBackup is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockRestoreProvider_Expecter) DeleteBackup(ctx interface{}, backup interface{}) *mockRestoreProvider_DeleteBackup_Call {
	return &mockRestoreProvider_DeleteBackup_Call{Call: _e.mock.On("DeleteBackup", ctx, backup)}
}

func (_c *mockRestoreProvider_DeleteBackup_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockRestoreProvider_DeleteBackup_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockRestoreProvider_DeleteBackup_Call) Return(_a0 error) *mockRestoreProvider_DeleteBackup_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRestoreProvider_DeleteBackup_Call) RunAndReturn(run func(context.Context, *v1.Backup) error) *mockRestoreProvider_DeleteBackup_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteRestore provides a mock function with given fields: ctx, restore
func (_m *mockRestoreProvider) DeleteRestore(ctx context.Context, restore *v1.Restore) error {
	ret := _m.Called(ctx, restore)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Restore) error); ok {
		r0 = rf(ctx, restore)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRestoreProvider_DeleteRestore_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteRestore'
type mockRestoreProvider_DeleteRestore_Call struct {
	*mock.Call
}

// DeleteRestore is a helper method to define mock.On call
//   - ctx context.Context
//   - restore *v1.Restore
func (_e *mockRestoreProvider_Expecter) DeleteRestore(ctx interface{}, restore interface{}) *mockRestoreProvider_DeleteRestore_Call {
	return &mockRestoreProvider_DeleteRestore_Call{Call: _e.mock.On("DeleteRestore", ctx, restore)}
}

func (_c *mockRestoreProvider_DeleteRestore_Call) Run(run func(ctx context.Context, restore *v1.Restore)) *mockRestoreProvider_DeleteRestore_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Restore))
	})
	return _c
}

func (_c *mockRestoreProvider_DeleteRestore_Call) Return(_a0 error) *mockRestoreProvider_DeleteRestore_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRestoreProvider_DeleteRestore_Call) RunAndReturn(run func(context.Context, *v1.Restore) error) *mockRestoreProvider_DeleteRestore_Call {
	_c.Call.Return(run)
	return _c
}

// SyncBackups provides a mock function with given fields: ctx
func (_m *mockRestoreProvider) SyncBackups(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockRestoreProvider_SyncBackups_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SyncBackups'
type mockRestoreProvider_SyncBackups_Call struct {
	*mock.Call
}

// SyncBackups is a helper method to define mock.On call
//   - ctx context.Context
func (_e *mockRestoreProvider_Expecter) SyncBackups(ctx interface{}) *mockRestoreProvider_SyncBackups_Call {
	return &mockRestoreProvider_SyncBackups_Call{Call: _e.mock.On("SyncBackups", ctx)}
}

func (_c *mockRestoreProvider_SyncBackups_Call) Run(run func(ctx context.Context)) *mockRestoreProvider_SyncBackups_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *mockRestoreProvider_SyncBackups_Call) Return(_a0 error) *mockRestoreProvider_SyncBackups_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRestoreProvider_SyncBackups_Call) RunAndReturn(run func(context.Context) error) *mockRestoreProvider_SyncBackups_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockRestoreProvider interface {
	mock.TestingT
	Cleanup(func())
}

// newMockRestoreProvider creates a new instance of mockRestoreProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockRestoreProvider(t mockConstructorTestingTnewMockRestoreProvider) *mockRestoreProvider {
	mock := &mockRestoreProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
