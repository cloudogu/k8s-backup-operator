// Code generated by mockery v2.20.0. DO NOT EDIT.

package backup

import (
	context "context"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// mockBackupProvider is an autogenerated mock type for the backupProvider type
type mockBackupProvider struct {
	mock.Mock
}

type mockBackupProvider_Expecter struct {
	mock *mock.Mock
}

func (_m *mockBackupProvider) EXPECT() *mockBackupProvider_Expecter {
	return &mockBackupProvider_Expecter{mock: &_m.Mock}
}

// CheckReady provides a mock function with given fields: ctx
func (_m *mockBackupProvider) CheckReady(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockBackupProvider_CheckReady_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CheckReady'
type mockBackupProvider_CheckReady_Call struct {
	*mock.Call
}

// CheckReady is a helper method to define mock.On call
//   - ctx context.Context
func (_e *mockBackupProvider_Expecter) CheckReady(ctx interface{}) *mockBackupProvider_CheckReady_Call {
	return &mockBackupProvider_CheckReady_Call{Call: _e.mock.On("CheckReady", ctx)}
}

func (_c *mockBackupProvider_CheckReady_Call) Run(run func(ctx context.Context)) *mockBackupProvider_CheckReady_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *mockBackupProvider_CheckReady_Call) Return(_a0 error) *mockBackupProvider_CheckReady_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockBackupProvider_CheckReady_Call) RunAndReturn(run func(context.Context) error) *mockBackupProvider_CheckReady_Call {
	_c.Call.Return(run)
	return _c
}

// CreateBackup provides a mock function with given fields: ctx, backup
func (_m *mockBackupProvider) CreateBackup(ctx context.Context, backup *v1.Backup) error {
	ret := _m.Called(ctx, backup)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) error); ok {
		r0 = rf(ctx, backup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockBackupProvider_CreateBackup_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateBackup'
type mockBackupProvider_CreateBackup_Call struct {
	*mock.Call
}

// CreateBackup is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockBackupProvider_Expecter) CreateBackup(ctx interface{}, backup interface{}) *mockBackupProvider_CreateBackup_Call {
	return &mockBackupProvider_CreateBackup_Call{Call: _e.mock.On("CreateBackup", ctx, backup)}
}

func (_c *mockBackupProvider_CreateBackup_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockBackupProvider_CreateBackup_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockBackupProvider_CreateBackup_Call) Return(_a0 error) *mockBackupProvider_CreateBackup_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockBackupProvider_CreateBackup_Call) RunAndReturn(run func(context.Context, *v1.Backup) error) *mockBackupProvider_CreateBackup_Call {
	_c.Call.Return(run)
	return _c
}

// CreateRestore provides a mock function with given fields: ctx, restore
func (_m *mockBackupProvider) CreateRestore(ctx context.Context, restore *v1.Restore) error {
	ret := _m.Called(ctx, restore)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Restore) error); ok {
		r0 = rf(ctx, restore)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockBackupProvider_CreateRestore_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CreateRestore'
type mockBackupProvider_CreateRestore_Call struct {
	*mock.Call
}

// CreateRestore is a helper method to define mock.On call
//   - ctx context.Context
//   - restore *v1.Restore
func (_e *mockBackupProvider_Expecter) CreateRestore(ctx interface{}, restore interface{}) *mockBackupProvider_CreateRestore_Call {
	return &mockBackupProvider_CreateRestore_Call{Call: _e.mock.On("CreateRestore", ctx, restore)}
}

func (_c *mockBackupProvider_CreateRestore_Call) Run(run func(ctx context.Context, restore *v1.Restore)) *mockBackupProvider_CreateRestore_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Restore))
	})
	return _c
}

func (_c *mockBackupProvider_CreateRestore_Call) Return(_a0 error) *mockBackupProvider_CreateRestore_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockBackupProvider_CreateRestore_Call) RunAndReturn(run func(context.Context, *v1.Restore) error) *mockBackupProvider_CreateRestore_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteBackup provides a mock function with given fields: ctx, backup
func (_m *mockBackupProvider) DeleteBackup(ctx context.Context, backup *v1.Backup) error {
	ret := _m.Called(ctx, backup)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) error); ok {
		r0 = rf(ctx, backup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockBackupProvider_DeleteBackup_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteBackup'
type mockBackupProvider_DeleteBackup_Call struct {
	*mock.Call
}

// DeleteBackup is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockBackupProvider_Expecter) DeleteBackup(ctx interface{}, backup interface{}) *mockBackupProvider_DeleteBackup_Call {
	return &mockBackupProvider_DeleteBackup_Call{Call: _e.mock.On("DeleteBackup", ctx, backup)}
}

func (_c *mockBackupProvider_DeleteBackup_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockBackupProvider_DeleteBackup_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockBackupProvider_DeleteBackup_Call) Return(_a0 error) *mockBackupProvider_DeleteBackup_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockBackupProvider_DeleteBackup_Call) RunAndReturn(run func(context.Context, *v1.Backup) error) *mockBackupProvider_DeleteBackup_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockBackupProvider interface {
	mock.TestingT
	Cleanup(func())
}

// newMockBackupProvider creates a new instance of mockBackupProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockBackupProvider(t mockConstructorTestingTnewMockBackupProvider) *mockBackupProvider {
	mock := &mockBackupProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
