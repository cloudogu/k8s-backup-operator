// Code generated by mockery v2.42.1. DO NOT EDIT.

package velero

import (
	mock "github.com/stretchr/testify/mock"
	rest "k8s.io/client-go/rest"

	v1 "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/typed/velero/v1"
)

// mockVeleroInterface is an autogenerated mock type for the veleroInterface type
type mockVeleroInterface struct {
	mock.Mock
}

type mockVeleroInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockVeleroInterface) EXPECT() *mockVeleroInterface_Expecter {
	return &mockVeleroInterface_Expecter{mock: &_m.Mock}
}

// BackupRepositories provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) BackupRepositories(namespace string) v1.BackupRepositoryInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for BackupRepositories")
	}

	var r0 v1.BackupRepositoryInterface
	if rf, ok := ret.Get(0).(func(string) v1.BackupRepositoryInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.BackupRepositoryInterface)
		}
	}

	return r0
}

// mockVeleroInterface_BackupRepositories_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BackupRepositories'
type mockVeleroInterface_BackupRepositories_Call struct {
	*mock.Call
}

// BackupRepositories is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) BackupRepositories(namespace interface{}) *mockVeleroInterface_BackupRepositories_Call {
	return &mockVeleroInterface_BackupRepositories_Call{Call: _e.mock.On("BackupRepositories", namespace)}
}

func (_c *mockVeleroInterface_BackupRepositories_Call) Run(run func(namespace string)) *mockVeleroInterface_BackupRepositories_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_BackupRepositories_Call) Return(_a0 v1.BackupRepositoryInterface) *mockVeleroInterface_BackupRepositories_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_BackupRepositories_Call) RunAndReturn(run func(string) v1.BackupRepositoryInterface) *mockVeleroInterface_BackupRepositories_Call {
	_c.Call.Return(run)
	return _c
}

// BackupStorageLocations provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) BackupStorageLocations(namespace string) v1.BackupStorageLocationInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for BackupStorageLocations")
	}

	var r0 v1.BackupStorageLocationInterface
	if rf, ok := ret.Get(0).(func(string) v1.BackupStorageLocationInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.BackupStorageLocationInterface)
		}
	}

	return r0
}

// mockVeleroInterface_BackupStorageLocations_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BackupStorageLocations'
type mockVeleroInterface_BackupStorageLocations_Call struct {
	*mock.Call
}

// BackupStorageLocations is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) BackupStorageLocations(namespace interface{}) *mockVeleroInterface_BackupStorageLocations_Call {
	return &mockVeleroInterface_BackupStorageLocations_Call{Call: _e.mock.On("BackupStorageLocations", namespace)}
}

func (_c *mockVeleroInterface_BackupStorageLocations_Call) Run(run func(namespace string)) *mockVeleroInterface_BackupStorageLocations_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_BackupStorageLocations_Call) Return(_a0 v1.BackupStorageLocationInterface) *mockVeleroInterface_BackupStorageLocations_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_BackupStorageLocations_Call) RunAndReturn(run func(string) v1.BackupStorageLocationInterface) *mockVeleroInterface_BackupStorageLocations_Call {
	_c.Call.Return(run)
	return _c
}

// Backups provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) Backups(namespace string) v1.BackupInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for Backups")
	}

	var r0 v1.BackupInterface
	if rf, ok := ret.Get(0).(func(string) v1.BackupInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.BackupInterface)
		}
	}

	return r0
}

// mockVeleroInterface_Backups_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Backups'
type mockVeleroInterface_Backups_Call struct {
	*mock.Call
}

// Backups is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) Backups(namespace interface{}) *mockVeleroInterface_Backups_Call {
	return &mockVeleroInterface_Backups_Call{Call: _e.mock.On("Backups", namespace)}
}

func (_c *mockVeleroInterface_Backups_Call) Run(run func(namespace string)) *mockVeleroInterface_Backups_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_Backups_Call) Return(_a0 v1.BackupInterface) *mockVeleroInterface_Backups_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_Backups_Call) RunAndReturn(run func(string) v1.BackupInterface) *mockVeleroInterface_Backups_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteBackupRequests provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) DeleteBackupRequests(namespace string) v1.DeleteBackupRequestInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for DeleteBackupRequests")
	}

	var r0 v1.DeleteBackupRequestInterface
	if rf, ok := ret.Get(0).(func(string) v1.DeleteBackupRequestInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.DeleteBackupRequestInterface)
		}
	}

	return r0
}

// mockVeleroInterface_DeleteBackupRequests_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteBackupRequests'
type mockVeleroInterface_DeleteBackupRequests_Call struct {
	*mock.Call
}

// DeleteBackupRequests is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) DeleteBackupRequests(namespace interface{}) *mockVeleroInterface_DeleteBackupRequests_Call {
	return &mockVeleroInterface_DeleteBackupRequests_Call{Call: _e.mock.On("DeleteBackupRequests", namespace)}
}

func (_c *mockVeleroInterface_DeleteBackupRequests_Call) Run(run func(namespace string)) *mockVeleroInterface_DeleteBackupRequests_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_DeleteBackupRequests_Call) Return(_a0 v1.DeleteBackupRequestInterface) *mockVeleroInterface_DeleteBackupRequests_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_DeleteBackupRequests_Call) RunAndReturn(run func(string) v1.DeleteBackupRequestInterface) *mockVeleroInterface_DeleteBackupRequests_Call {
	_c.Call.Return(run)
	return _c
}

// DownloadRequests provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) DownloadRequests(namespace string) v1.DownloadRequestInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for DownloadRequests")
	}

	var r0 v1.DownloadRequestInterface
	if rf, ok := ret.Get(0).(func(string) v1.DownloadRequestInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.DownloadRequestInterface)
		}
	}

	return r0
}

// mockVeleroInterface_DownloadRequests_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DownloadRequests'
type mockVeleroInterface_DownloadRequests_Call struct {
	*mock.Call
}

// DownloadRequests is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) DownloadRequests(namespace interface{}) *mockVeleroInterface_DownloadRequests_Call {
	return &mockVeleroInterface_DownloadRequests_Call{Call: _e.mock.On("DownloadRequests", namespace)}
}

func (_c *mockVeleroInterface_DownloadRequests_Call) Run(run func(namespace string)) *mockVeleroInterface_DownloadRequests_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_DownloadRequests_Call) Return(_a0 v1.DownloadRequestInterface) *mockVeleroInterface_DownloadRequests_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_DownloadRequests_Call) RunAndReturn(run func(string) v1.DownloadRequestInterface) *mockVeleroInterface_DownloadRequests_Call {
	_c.Call.Return(run)
	return _c
}

// PodVolumeBackups provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) PodVolumeBackups(namespace string) v1.PodVolumeBackupInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for PodVolumeBackups")
	}

	var r0 v1.PodVolumeBackupInterface
	if rf, ok := ret.Get(0).(func(string) v1.PodVolumeBackupInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.PodVolumeBackupInterface)
		}
	}

	return r0
}

// mockVeleroInterface_PodVolumeBackups_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PodVolumeBackups'
type mockVeleroInterface_PodVolumeBackups_Call struct {
	*mock.Call
}

// PodVolumeBackups is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) PodVolumeBackups(namespace interface{}) *mockVeleroInterface_PodVolumeBackups_Call {
	return &mockVeleroInterface_PodVolumeBackups_Call{Call: _e.mock.On("PodVolumeBackups", namespace)}
}

func (_c *mockVeleroInterface_PodVolumeBackups_Call) Run(run func(namespace string)) *mockVeleroInterface_PodVolumeBackups_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_PodVolumeBackups_Call) Return(_a0 v1.PodVolumeBackupInterface) *mockVeleroInterface_PodVolumeBackups_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_PodVolumeBackups_Call) RunAndReturn(run func(string) v1.PodVolumeBackupInterface) *mockVeleroInterface_PodVolumeBackups_Call {
	_c.Call.Return(run)
	return _c
}

// PodVolumeRestores provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) PodVolumeRestores(namespace string) v1.PodVolumeRestoreInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for PodVolumeRestores")
	}

	var r0 v1.PodVolumeRestoreInterface
	if rf, ok := ret.Get(0).(func(string) v1.PodVolumeRestoreInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.PodVolumeRestoreInterface)
		}
	}

	return r0
}

// mockVeleroInterface_PodVolumeRestores_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PodVolumeRestores'
type mockVeleroInterface_PodVolumeRestores_Call struct {
	*mock.Call
}

// PodVolumeRestores is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) PodVolumeRestores(namespace interface{}) *mockVeleroInterface_PodVolumeRestores_Call {
	return &mockVeleroInterface_PodVolumeRestores_Call{Call: _e.mock.On("PodVolumeRestores", namespace)}
}

func (_c *mockVeleroInterface_PodVolumeRestores_Call) Run(run func(namespace string)) *mockVeleroInterface_PodVolumeRestores_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_PodVolumeRestores_Call) Return(_a0 v1.PodVolumeRestoreInterface) *mockVeleroInterface_PodVolumeRestores_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_PodVolumeRestores_Call) RunAndReturn(run func(string) v1.PodVolumeRestoreInterface) *mockVeleroInterface_PodVolumeRestores_Call {
	_c.Call.Return(run)
	return _c
}

// RESTClient provides a mock function with given fields:
func (_m *mockVeleroInterface) RESTClient() rest.Interface {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for RESTClient")
	}

	var r0 rest.Interface
	if rf, ok := ret.Get(0).(func() rest.Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rest.Interface)
		}
	}

	return r0
}

// mockVeleroInterface_RESTClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RESTClient'
type mockVeleroInterface_RESTClient_Call struct {
	*mock.Call
}

// RESTClient is a helper method to define mock.On call
func (_e *mockVeleroInterface_Expecter) RESTClient() *mockVeleroInterface_RESTClient_Call {
	return &mockVeleroInterface_RESTClient_Call{Call: _e.mock.On("RESTClient")}
}

func (_c *mockVeleroInterface_RESTClient_Call) Run(run func()) *mockVeleroInterface_RESTClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockVeleroInterface_RESTClient_Call) Return(_a0 rest.Interface) *mockVeleroInterface_RESTClient_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_RESTClient_Call) RunAndReturn(run func() rest.Interface) *mockVeleroInterface_RESTClient_Call {
	_c.Call.Return(run)
	return _c
}

// Restores provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) Restores(namespace string) v1.RestoreInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for Restores")
	}

	var r0 v1.RestoreInterface
	if rf, ok := ret.Get(0).(func(string) v1.RestoreInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.RestoreInterface)
		}
	}

	return r0
}

// mockVeleroInterface_Restores_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Restores'
type mockVeleroInterface_Restores_Call struct {
	*mock.Call
}

// Restores is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) Restores(namespace interface{}) *mockVeleroInterface_Restores_Call {
	return &mockVeleroInterface_Restores_Call{Call: _e.mock.On("Restores", namespace)}
}

func (_c *mockVeleroInterface_Restores_Call) Run(run func(namespace string)) *mockVeleroInterface_Restores_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_Restores_Call) Return(_a0 v1.RestoreInterface) *mockVeleroInterface_Restores_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_Restores_Call) RunAndReturn(run func(string) v1.RestoreInterface) *mockVeleroInterface_Restores_Call {
	_c.Call.Return(run)
	return _c
}

// Schedules provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) Schedules(namespace string) v1.ScheduleInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for Schedules")
	}

	var r0 v1.ScheduleInterface
	if rf, ok := ret.Get(0).(func(string) v1.ScheduleInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.ScheduleInterface)
		}
	}

	return r0
}

// mockVeleroInterface_Schedules_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Schedules'
type mockVeleroInterface_Schedules_Call struct {
	*mock.Call
}

// Schedules is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) Schedules(namespace interface{}) *mockVeleroInterface_Schedules_Call {
	return &mockVeleroInterface_Schedules_Call{Call: _e.mock.On("Schedules", namespace)}
}

func (_c *mockVeleroInterface_Schedules_Call) Run(run func(namespace string)) *mockVeleroInterface_Schedules_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_Schedules_Call) Return(_a0 v1.ScheduleInterface) *mockVeleroInterface_Schedules_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_Schedules_Call) RunAndReturn(run func(string) v1.ScheduleInterface) *mockVeleroInterface_Schedules_Call {
	_c.Call.Return(run)
	return _c
}

// ServerStatusRequests provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) ServerStatusRequests(namespace string) v1.ServerStatusRequestInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for ServerStatusRequests")
	}

	var r0 v1.ServerStatusRequestInterface
	if rf, ok := ret.Get(0).(func(string) v1.ServerStatusRequestInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.ServerStatusRequestInterface)
		}
	}

	return r0
}

// mockVeleroInterface_ServerStatusRequests_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ServerStatusRequests'
type mockVeleroInterface_ServerStatusRequests_Call struct {
	*mock.Call
}

// ServerStatusRequests is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) ServerStatusRequests(namespace interface{}) *mockVeleroInterface_ServerStatusRequests_Call {
	return &mockVeleroInterface_ServerStatusRequests_Call{Call: _e.mock.On("ServerStatusRequests", namespace)}
}

func (_c *mockVeleroInterface_ServerStatusRequests_Call) Run(run func(namespace string)) *mockVeleroInterface_ServerStatusRequests_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_ServerStatusRequests_Call) Return(_a0 v1.ServerStatusRequestInterface) *mockVeleroInterface_ServerStatusRequests_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_ServerStatusRequests_Call) RunAndReturn(run func(string) v1.ServerStatusRequestInterface) *mockVeleroInterface_ServerStatusRequests_Call {
	_c.Call.Return(run)
	return _c
}

// VolumeSnapshotLocations provides a mock function with given fields: namespace
func (_m *mockVeleroInterface) VolumeSnapshotLocations(namespace string) v1.VolumeSnapshotLocationInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for VolumeSnapshotLocations")
	}

	var r0 v1.VolumeSnapshotLocationInterface
	if rf, ok := ret.Get(0).(func(string) v1.VolumeSnapshotLocationInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.VolumeSnapshotLocationInterface)
		}
	}

	return r0
}

// mockVeleroInterface_VolumeSnapshotLocations_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'VolumeSnapshotLocations'
type mockVeleroInterface_VolumeSnapshotLocations_Call struct {
	*mock.Call
}

// VolumeSnapshotLocations is a helper method to define mock.On call
//   - namespace string
func (_e *mockVeleroInterface_Expecter) VolumeSnapshotLocations(namespace interface{}) *mockVeleroInterface_VolumeSnapshotLocations_Call {
	return &mockVeleroInterface_VolumeSnapshotLocations_Call{Call: _e.mock.On("VolumeSnapshotLocations", namespace)}
}

func (_c *mockVeleroInterface_VolumeSnapshotLocations_Call) Run(run func(namespace string)) *mockVeleroInterface_VolumeSnapshotLocations_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockVeleroInterface_VolumeSnapshotLocations_Call) Return(_a0 v1.VolumeSnapshotLocationInterface) *mockVeleroInterface_VolumeSnapshotLocations_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroInterface_VolumeSnapshotLocations_Call) RunAndReturn(run func(string) v1.VolumeSnapshotLocationInterface) *mockVeleroInterface_VolumeSnapshotLocations_Call {
	_c.Call.Return(run)
	return _c
}

// newMockVeleroInterface creates a new instance of mockVeleroInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockVeleroInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockVeleroInterface {
	mock := &mockVeleroInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
