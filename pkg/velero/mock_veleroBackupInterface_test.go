// Code generated by mockery v2.53.3. DO NOT EDIT.

package velero

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	types "k8s.io/apimachinery/pkg/types"

	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// mockVeleroBackupInterface is an autogenerated mock type for the veleroBackupInterface type
type mockVeleroBackupInterface struct {
	mock.Mock
}

type mockVeleroBackupInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockVeleroBackupInterface) EXPECT() *mockVeleroBackupInterface_Expecter {
	return &mockVeleroBackupInterface_Expecter{mock: &_m.Mock}
}

// Create provides a mock function with given fields: ctx, backup, opts
func (_m *mockVeleroBackupInterface) Create(ctx context.Context, backup *v1.Backup, opts metav1.CreateOptions) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup, opts)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, metav1.CreateOptions) (*v1.Backup, error)); ok {
		return rf(ctx, backup, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, metav1.CreateOptions) *v1.Backup); ok {
		r0 = rf(ctx, backup, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, backup, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockVeleroBackupInterface_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type mockVeleroBackupInterface_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - opts metav1.CreateOptions
func (_e *mockVeleroBackupInterface_Expecter) Create(ctx interface{}, backup interface{}, opts interface{}) *mockVeleroBackupInterface_Create_Call {
	return &mockVeleroBackupInterface_Create_Call{Call: _e.mock.On("Create", ctx, backup, opts)}
}

func (_c *mockVeleroBackupInterface_Create_Call) Run(run func(ctx context.Context, backup *v1.Backup, opts metav1.CreateOptions)) *mockVeleroBackupInterface_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_Create_Call) Return(_a0 *v1.Backup, _a1 error) *mockVeleroBackupInterface_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockVeleroBackupInterface_Create_Call) RunAndReturn(run func(context.Context, *v1.Backup, metav1.CreateOptions) (*v1.Backup, error)) *mockVeleroBackupInterface_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *mockVeleroBackupInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	if len(ret) == 0 {
		panic("no return value specified for Delete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockVeleroBackupInterface_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type mockVeleroBackupInterface_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.DeleteOptions
func (_e *mockVeleroBackupInterface_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *mockVeleroBackupInterface_Delete_Call {
	return &mockVeleroBackupInterface_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *mockVeleroBackupInterface_Delete_Call) Run(run func(ctx context.Context, name string, opts metav1.DeleteOptions)) *mockVeleroBackupInterface_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.DeleteOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_Delete_Call) Return(_a0 error) *mockVeleroBackupInterface_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroBackupInterface_Delete_Call) RunAndReturn(run func(context.Context, string, metav1.DeleteOptions) error) *mockVeleroBackupInterface_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *mockVeleroBackupInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	if len(ret) == 0 {
		panic("no return value specified for DeleteCollection")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockVeleroBackupInterface_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type mockVeleroBackupInterface_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.DeleteOptions
//   - listOpts metav1.ListOptions
func (_e *mockVeleroBackupInterface_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *mockVeleroBackupInterface_DeleteCollection_Call {
	return &mockVeleroBackupInterface_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *mockVeleroBackupInterface_DeleteCollection_Call) Run(run func(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions)) *mockVeleroBackupInterface_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.DeleteOptions), args[2].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_DeleteCollection_Call) Return(_a0 error) *mockVeleroBackupInterface_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockVeleroBackupInterface_DeleteCollection_Call) RunAndReturn(run func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) *mockVeleroBackupInterface_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *mockVeleroBackupInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Backup, error) {
	ret := _m.Called(ctx, name, opts)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) (*v1.Backup, error)); ok {
		return rf(ctx, name, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *v1.Backup); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockVeleroBackupInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockVeleroBackupInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.GetOptions
func (_e *mockVeleroBackupInterface_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *mockVeleroBackupInterface_Get_Call {
	return &mockVeleroBackupInterface_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *mockVeleroBackupInterface_Get_Call) Run(run func(ctx context.Context, name string, opts metav1.GetOptions)) *mockVeleroBackupInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_Get_Call) Return(_a0 *v1.Backup, _a1 error) *mockVeleroBackupInterface_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockVeleroBackupInterface_Get_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*v1.Backup, error)) *mockVeleroBackupInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *mockVeleroBackupInterface) List(ctx context.Context, opts metav1.ListOptions) (*v1.BackupList, error) {
	ret := _m.Called(ctx, opts)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 *v1.BackupList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (*v1.BackupList, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) *v1.BackupList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.BackupList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockVeleroBackupInterface_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type mockVeleroBackupInterface_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockVeleroBackupInterface_Expecter) List(ctx interface{}, opts interface{}) *mockVeleroBackupInterface_List_Call {
	return &mockVeleroBackupInterface_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *mockVeleroBackupInterface_List_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockVeleroBackupInterface_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_List_Call) Return(_a0 *v1.BackupList, _a1 error) *mockVeleroBackupInterface_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockVeleroBackupInterface_List_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (*v1.BackupList, error)) *mockVeleroBackupInterface_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *mockVeleroBackupInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.Backup, error) {
	_va := make([]interface{}, len(subresources))
	for _i := range subresources {
		_va[_i] = subresources[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name, pt, data, opts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Patch")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*v1.Backup, error)); ok {
		return rf(ctx, name, pt, data, opts, subresources...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) *v1.Backup); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockVeleroBackupInterface_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type mockVeleroBackupInterface_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts metav1.PatchOptions
//   - subresources ...string
func (_e *mockVeleroBackupInterface_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *mockVeleroBackupInterface_Patch_Call {
	return &mockVeleroBackupInterface_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *mockVeleroBackupInterface_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string)) *mockVeleroBackupInterface_Patch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]string, len(args)-5)
		for i, a := range args[5:] {
			if a != nil {
				variadicArgs[i] = a.(string)
			}
		}
		run(args[0].(context.Context), args[1].(string), args[2].(types.PatchType), args[3].([]byte), args[4].(metav1.PatchOptions), variadicArgs...)
	})
	return _c
}

func (_c *mockVeleroBackupInterface_Patch_Call) Return(result *v1.Backup, err error) *mockVeleroBackupInterface_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockVeleroBackupInterface_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*v1.Backup, error)) *mockVeleroBackupInterface_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, backup, opts
func (_m *mockVeleroBackupInterface) Update(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup, opts)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, metav1.UpdateOptions) (*v1.Backup, error)); ok {
		return rf(ctx, backup, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, metav1.UpdateOptions) *v1.Backup); ok {
		r0 = rf(ctx, backup, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, backup, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockVeleroBackupInterface_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockVeleroBackupInterface_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - opts metav1.UpdateOptions
func (_e *mockVeleroBackupInterface_Expecter) Update(ctx interface{}, backup interface{}, opts interface{}) *mockVeleroBackupInterface_Update_Call {
	return &mockVeleroBackupInterface_Update_Call{Call: _e.mock.On("Update", ctx, backup, opts)}
}

func (_c *mockVeleroBackupInterface_Update_Call) Run(run func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions)) *mockVeleroBackupInterface_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_Update_Call) Return(_a0 *v1.Backup, _a1 error) *mockVeleroBackupInterface_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockVeleroBackupInterface_Update_Call) RunAndReturn(run func(context.Context, *v1.Backup, metav1.UpdateOptions) (*v1.Backup, error)) *mockVeleroBackupInterface_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, backup, opts
func (_m *mockVeleroBackupInterface) UpdateStatus(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup, opts)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatus")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, metav1.UpdateOptions) (*v1.Backup, error)); ok {
		return rf(ctx, backup, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, metav1.UpdateOptions) *v1.Backup); ok {
		r0 = rf(ctx, backup, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, backup, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockVeleroBackupInterface_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type mockVeleroBackupInterface_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - opts metav1.UpdateOptions
func (_e *mockVeleroBackupInterface_Expecter) UpdateStatus(ctx interface{}, backup interface{}, opts interface{}) *mockVeleroBackupInterface_UpdateStatus_Call {
	return &mockVeleroBackupInterface_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, backup, opts)}
}

func (_c *mockVeleroBackupInterface_UpdateStatus_Call) Run(run func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions)) *mockVeleroBackupInterface_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_UpdateStatus_Call) Return(_a0 *v1.Backup, _a1 error) *mockVeleroBackupInterface_UpdateStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockVeleroBackupInterface_UpdateStatus_Call) RunAndReturn(run func(context.Context, *v1.Backup, metav1.UpdateOptions) (*v1.Backup, error)) *mockVeleroBackupInterface_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *mockVeleroBackupInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	ret := _m.Called(ctx, opts)

	if len(ret) == 0 {
		panic("no return value specified for Watch")
	}

	var r0 watch.Interface
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (watch.Interface, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) watch.Interface); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(watch.Interface)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockVeleroBackupInterface_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type mockVeleroBackupInterface_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockVeleroBackupInterface_Expecter) Watch(ctx interface{}, opts interface{}) *mockVeleroBackupInterface_Watch_Call {
	return &mockVeleroBackupInterface_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *mockVeleroBackupInterface_Watch_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockVeleroBackupInterface_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockVeleroBackupInterface_Watch_Call) Return(_a0 watch.Interface, _a1 error) *mockVeleroBackupInterface_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockVeleroBackupInterface_Watch_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (watch.Interface, error)) *mockVeleroBackupInterface_Watch_Call {
	_c.Call.Return(run)
	return _c
}

// newMockVeleroBackupInterface creates a new instance of mockVeleroBackupInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockVeleroBackupInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockVeleroBackupInterface {
	mock := &mockVeleroBackupInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
