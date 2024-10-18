// Code generated by mockery v2.42.1. DO NOT EDIT.

package backup

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	types "k8s.io/apimachinery/pkg/types"

	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// mockEcosystemBackupInterface is an autogenerated mock type for the ecosystemBackupInterface type
type mockEcosystemBackupInterface struct {
	mock.Mock
}

type mockEcosystemBackupInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockEcosystemBackupInterface) EXPECT() *mockEcosystemBackupInterface_Expecter {
	return &mockEcosystemBackupInterface_Expecter{mock: &_m.Mock}
}

// AddFinalizer provides a mock function with given fields: ctx, backup, finalizer
func (_m *mockEcosystemBackupInterface) AddFinalizer(ctx context.Context, backup *v1.Backup, finalizer string) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup, finalizer)

	if len(ret) == 0 {
		panic("no return value specified for AddFinalizer")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, string) (*v1.Backup, error)); ok {
		return rf(ctx, backup, finalizer)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, string) *v1.Backup); ok {
		r0 = rf(ctx, backup, finalizer)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup, string) error); ok {
		r1 = rf(ctx, backup, finalizer)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEcosystemBackupInterface_AddFinalizer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddFinalizer'
type mockEcosystemBackupInterface_AddFinalizer_Call struct {
	*mock.Call
}

// AddFinalizer is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - finalizer string
func (_e *mockEcosystemBackupInterface_Expecter) AddFinalizer(ctx interface{}, backup interface{}, finalizer interface{}) *mockEcosystemBackupInterface_AddFinalizer_Call {
	return &mockEcosystemBackupInterface_AddFinalizer_Call{Call: _e.mock.On("AddFinalizer", ctx, backup, finalizer)}
}

func (_c *mockEcosystemBackupInterface_AddFinalizer_Call) Run(run func(ctx context.Context, backup *v1.Backup, finalizer string)) *mockEcosystemBackupInterface_AddFinalizer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(string))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_AddFinalizer_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_AddFinalizer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_AddFinalizer_Call) RunAndReturn(run func(context.Context, *v1.Backup, string) (*v1.Backup, error)) *mockEcosystemBackupInterface_AddFinalizer_Call {
	_c.Call.Return(run)
	return _c
}

// AddLabels provides a mock function with given fields: ctx, backup
func (_m *mockEcosystemBackupInterface) AddLabels(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup)

	if len(ret) == 0 {
		panic("no return value specified for AddLabels")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) (*v1.Backup, error)); ok {
		return rf(ctx, backup)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) *v1.Backup); ok {
		r0 = rf(ctx, backup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup) error); ok {
		r1 = rf(ctx, backup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEcosystemBackupInterface_AddLabels_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AddLabels'
type mockEcosystemBackupInterface_AddLabels_Call struct {
	*mock.Call
}

// AddLabels is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockEcosystemBackupInterface_Expecter) AddLabels(ctx interface{}, backup interface{}) *mockEcosystemBackupInterface_AddLabels_Call {
	return &mockEcosystemBackupInterface_AddLabels_Call{Call: _e.mock.On("AddLabels", ctx, backup)}
}

func (_c *mockEcosystemBackupInterface_AddLabels_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockEcosystemBackupInterface_AddLabels_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_AddLabels_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_AddLabels_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_AddLabels_Call) RunAndReturn(run func(context.Context, *v1.Backup) (*v1.Backup, error)) *mockEcosystemBackupInterface_AddLabels_Call {
	_c.Call.Return(run)
	return _c
}

// Create provides a mock function with given fields: ctx, backup, opts
func (_m *mockEcosystemBackupInterface) Create(ctx context.Context, backup *v1.Backup, opts metav1.CreateOptions) (*v1.Backup, error) {
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

// mockEcosystemBackupInterface_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type mockEcosystemBackupInterface_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - opts metav1.CreateOptions
func (_e *mockEcosystemBackupInterface_Expecter) Create(ctx interface{}, backup interface{}, opts interface{}) *mockEcosystemBackupInterface_Create_Call {
	return &mockEcosystemBackupInterface_Create_Call{Call: _e.mock.On("Create", ctx, backup, opts)}
}

func (_c *mockEcosystemBackupInterface_Create_Call) Run(run func(ctx context.Context, backup *v1.Backup, opts metav1.CreateOptions)) *mockEcosystemBackupInterface_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_Create_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_Create_Call) RunAndReturn(run func(context.Context, *v1.Backup, metav1.CreateOptions) (*v1.Backup, error)) *mockEcosystemBackupInterface_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *mockEcosystemBackupInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
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

// mockEcosystemBackupInterface_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type mockEcosystemBackupInterface_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.DeleteOptions
func (_e *mockEcosystemBackupInterface_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *mockEcosystemBackupInterface_Delete_Call {
	return &mockEcosystemBackupInterface_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *mockEcosystemBackupInterface_Delete_Call) Run(run func(ctx context.Context, name string, opts metav1.DeleteOptions)) *mockEcosystemBackupInterface_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.DeleteOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_Delete_Call) Return(_a0 error) *mockEcosystemBackupInterface_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEcosystemBackupInterface_Delete_Call) RunAndReturn(run func(context.Context, string, metav1.DeleteOptions) error) *mockEcosystemBackupInterface_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *mockEcosystemBackupInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
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

// mockEcosystemBackupInterface_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type mockEcosystemBackupInterface_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.DeleteOptions
//   - listOpts metav1.ListOptions
func (_e *mockEcosystemBackupInterface_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *mockEcosystemBackupInterface_DeleteCollection_Call {
	return &mockEcosystemBackupInterface_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *mockEcosystemBackupInterface_DeleteCollection_Call) Run(run func(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions)) *mockEcosystemBackupInterface_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.DeleteOptions), args[2].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_DeleteCollection_Call) Return(_a0 error) *mockEcosystemBackupInterface_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEcosystemBackupInterface_DeleteCollection_Call) RunAndReturn(run func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) *mockEcosystemBackupInterface_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *mockEcosystemBackupInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Backup, error) {
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

// mockEcosystemBackupInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockEcosystemBackupInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.GetOptions
func (_e *mockEcosystemBackupInterface_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *mockEcosystemBackupInterface_Get_Call {
	return &mockEcosystemBackupInterface_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *mockEcosystemBackupInterface_Get_Call) Run(run func(ctx context.Context, name string, opts metav1.GetOptions)) *mockEcosystemBackupInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_Get_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_Get_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*v1.Backup, error)) *mockEcosystemBackupInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *mockEcosystemBackupInterface) List(ctx context.Context, opts metav1.ListOptions) (*v1.BackupList, error) {
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

// mockEcosystemBackupInterface_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type mockEcosystemBackupInterface_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockEcosystemBackupInterface_Expecter) List(ctx interface{}, opts interface{}) *mockEcosystemBackupInterface_List_Call {
	return &mockEcosystemBackupInterface_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *mockEcosystemBackupInterface_List_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockEcosystemBackupInterface_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_List_Call) Return(_a0 *v1.BackupList, _a1 error) *mockEcosystemBackupInterface_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_List_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (*v1.BackupList, error)) *mockEcosystemBackupInterface_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *mockEcosystemBackupInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*v1.Backup, error) {
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

// mockEcosystemBackupInterface_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type mockEcosystemBackupInterface_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts metav1.PatchOptions
//   - subresources ...string
func (_e *mockEcosystemBackupInterface_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *mockEcosystemBackupInterface_Patch_Call {
	return &mockEcosystemBackupInterface_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *mockEcosystemBackupInterface_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string)) *mockEcosystemBackupInterface_Patch_Call {
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

func (_c *mockEcosystemBackupInterface_Patch_Call) Return(result *v1.Backup, err error) *mockEcosystemBackupInterface_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockEcosystemBackupInterface_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*v1.Backup, error)) *mockEcosystemBackupInterface_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveFinalizer provides a mock function with given fields: ctx, backup, finalizer
func (_m *mockEcosystemBackupInterface) RemoveFinalizer(ctx context.Context, backup *v1.Backup, finalizer string) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup, finalizer)

	if len(ret) == 0 {
		panic("no return value specified for RemoveFinalizer")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, string) (*v1.Backup, error)); ok {
		return rf(ctx, backup, finalizer)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup, string) *v1.Backup); ok {
		r0 = rf(ctx, backup, finalizer)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup, string) error); ok {
		r1 = rf(ctx, backup, finalizer)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEcosystemBackupInterface_RemoveFinalizer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveFinalizer'
type mockEcosystemBackupInterface_RemoveFinalizer_Call struct {
	*mock.Call
}

// RemoveFinalizer is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - finalizer string
func (_e *mockEcosystemBackupInterface_Expecter) RemoveFinalizer(ctx interface{}, backup interface{}, finalizer interface{}) *mockEcosystemBackupInterface_RemoveFinalizer_Call {
	return &mockEcosystemBackupInterface_RemoveFinalizer_Call{Call: _e.mock.On("RemoveFinalizer", ctx, backup, finalizer)}
}

func (_c *mockEcosystemBackupInterface_RemoveFinalizer_Call) Run(run func(ctx context.Context, backup *v1.Backup, finalizer string)) *mockEcosystemBackupInterface_RemoveFinalizer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(string))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_RemoveFinalizer_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_RemoveFinalizer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_RemoveFinalizer_Call) RunAndReturn(run func(context.Context, *v1.Backup, string) (*v1.Backup, error)) *mockEcosystemBackupInterface_RemoveFinalizer_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, backup, opts
func (_m *mockEcosystemBackupInterface) Update(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (*v1.Backup, error) {
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

// mockEcosystemBackupInterface_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockEcosystemBackupInterface_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - opts metav1.UpdateOptions
func (_e *mockEcosystemBackupInterface_Expecter) Update(ctx interface{}, backup interface{}, opts interface{}) *mockEcosystemBackupInterface_Update_Call {
	return &mockEcosystemBackupInterface_Update_Call{Call: _e.mock.On("Update", ctx, backup, opts)}
}

func (_c *mockEcosystemBackupInterface_Update_Call) Run(run func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions)) *mockEcosystemBackupInterface_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_Update_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_Update_Call) RunAndReturn(run func(context.Context, *v1.Backup, metav1.UpdateOptions) (*v1.Backup, error)) *mockEcosystemBackupInterface_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, backup, opts
func (_m *mockEcosystemBackupInterface) UpdateStatus(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions) (*v1.Backup, error) {
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

// mockEcosystemBackupInterface_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type mockEcosystemBackupInterface_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
//   - opts metav1.UpdateOptions
func (_e *mockEcosystemBackupInterface_Expecter) UpdateStatus(ctx interface{}, backup interface{}, opts interface{}) *mockEcosystemBackupInterface_UpdateStatus_Call {
	return &mockEcosystemBackupInterface_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, backup, opts)}
}

func (_c *mockEcosystemBackupInterface_UpdateStatus_Call) Run(run func(ctx context.Context, backup *v1.Backup, opts metav1.UpdateOptions)) *mockEcosystemBackupInterface_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatus_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_UpdateStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatus_Call) RunAndReturn(run func(context.Context, *v1.Backup, metav1.UpdateOptions) (*v1.Backup, error)) *mockEcosystemBackupInterface_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatusCompleted provides a mock function with given fields: ctx, backup
func (_m *mockEcosystemBackupInterface) UpdateStatusCompleted(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatusCompleted")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) (*v1.Backup, error)); ok {
		return rf(ctx, backup)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) *v1.Backup); ok {
		r0 = rf(ctx, backup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup) error); ok {
		r1 = rf(ctx, backup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEcosystemBackupInterface_UpdateStatusCompleted_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatusCompleted'
type mockEcosystemBackupInterface_UpdateStatusCompleted_Call struct {
	*mock.Call
}

// UpdateStatusCompleted is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockEcosystemBackupInterface_Expecter) UpdateStatusCompleted(ctx interface{}, backup interface{}) *mockEcosystemBackupInterface_UpdateStatusCompleted_Call {
	return &mockEcosystemBackupInterface_UpdateStatusCompleted_Call{Call: _e.mock.On("UpdateStatusCompleted", ctx, backup)}
}

func (_c *mockEcosystemBackupInterface_UpdateStatusCompleted_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockEcosystemBackupInterface_UpdateStatusCompleted_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusCompleted_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_UpdateStatusCompleted_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusCompleted_Call) RunAndReturn(run func(context.Context, *v1.Backup) (*v1.Backup, error)) *mockEcosystemBackupInterface_UpdateStatusCompleted_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatusDeleting provides a mock function with given fields: ctx, backup
func (_m *mockEcosystemBackupInterface) UpdateStatusDeleting(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatusDeleting")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) (*v1.Backup, error)); ok {
		return rf(ctx, backup)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) *v1.Backup); ok {
		r0 = rf(ctx, backup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup) error); ok {
		r1 = rf(ctx, backup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEcosystemBackupInterface_UpdateStatusDeleting_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatusDeleting'
type mockEcosystemBackupInterface_UpdateStatusDeleting_Call struct {
	*mock.Call
}

// UpdateStatusDeleting is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockEcosystemBackupInterface_Expecter) UpdateStatusDeleting(ctx interface{}, backup interface{}) *mockEcosystemBackupInterface_UpdateStatusDeleting_Call {
	return &mockEcosystemBackupInterface_UpdateStatusDeleting_Call{Call: _e.mock.On("UpdateStatusDeleting", ctx, backup)}
}

func (_c *mockEcosystemBackupInterface_UpdateStatusDeleting_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockEcosystemBackupInterface_UpdateStatusDeleting_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusDeleting_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_UpdateStatusDeleting_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusDeleting_Call) RunAndReturn(run func(context.Context, *v1.Backup) (*v1.Backup, error)) *mockEcosystemBackupInterface_UpdateStatusDeleting_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatusFailed provides a mock function with given fields: ctx, backup
func (_m *mockEcosystemBackupInterface) UpdateStatusFailed(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatusFailed")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) (*v1.Backup, error)); ok {
		return rf(ctx, backup)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) *v1.Backup); ok {
		r0 = rf(ctx, backup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup) error); ok {
		r1 = rf(ctx, backup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEcosystemBackupInterface_UpdateStatusFailed_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatusFailed'
type mockEcosystemBackupInterface_UpdateStatusFailed_Call struct {
	*mock.Call
}

// UpdateStatusFailed is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockEcosystemBackupInterface_Expecter) UpdateStatusFailed(ctx interface{}, backup interface{}) *mockEcosystemBackupInterface_UpdateStatusFailed_Call {
	return &mockEcosystemBackupInterface_UpdateStatusFailed_Call{Call: _e.mock.On("UpdateStatusFailed", ctx, backup)}
}

func (_c *mockEcosystemBackupInterface_UpdateStatusFailed_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockEcosystemBackupInterface_UpdateStatusFailed_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusFailed_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_UpdateStatusFailed_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusFailed_Call) RunAndReturn(run func(context.Context, *v1.Backup) (*v1.Backup, error)) *mockEcosystemBackupInterface_UpdateStatusFailed_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatusInProgress provides a mock function with given fields: ctx, backup
func (_m *mockEcosystemBackupInterface) UpdateStatusInProgress(ctx context.Context, backup *v1.Backup) (*v1.Backup, error) {
	ret := _m.Called(ctx, backup)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatusInProgress")
	}

	var r0 *v1.Backup
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) (*v1.Backup, error)); ok {
		return rf(ctx, backup)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.Backup) *v1.Backup); ok {
		r0 = rf(ctx, backup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Backup)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.Backup) error); ok {
		r1 = rf(ctx, backup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEcosystemBackupInterface_UpdateStatusInProgress_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatusInProgress'
type mockEcosystemBackupInterface_UpdateStatusInProgress_Call struct {
	*mock.Call
}

// UpdateStatusInProgress is a helper method to define mock.On call
//   - ctx context.Context
//   - backup *v1.Backup
func (_e *mockEcosystemBackupInterface_Expecter) UpdateStatusInProgress(ctx interface{}, backup interface{}) *mockEcosystemBackupInterface_UpdateStatusInProgress_Call {
	return &mockEcosystemBackupInterface_UpdateStatusInProgress_Call{Call: _e.mock.On("UpdateStatusInProgress", ctx, backup)}
}

func (_c *mockEcosystemBackupInterface_UpdateStatusInProgress_Call) Run(run func(ctx context.Context, backup *v1.Backup)) *mockEcosystemBackupInterface_UpdateStatusInProgress_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.Backup))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusInProgress_Call) Return(_a0 *v1.Backup, _a1 error) *mockEcosystemBackupInterface_UpdateStatusInProgress_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_UpdateStatusInProgress_Call) RunAndReturn(run func(context.Context, *v1.Backup) (*v1.Backup, error)) *mockEcosystemBackupInterface_UpdateStatusInProgress_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *mockEcosystemBackupInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
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

// mockEcosystemBackupInterface_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type mockEcosystemBackupInterface_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockEcosystemBackupInterface_Expecter) Watch(ctx interface{}, opts interface{}) *mockEcosystemBackupInterface_Watch_Call {
	return &mockEcosystemBackupInterface_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *mockEcosystemBackupInterface_Watch_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockEcosystemBackupInterface_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockEcosystemBackupInterface_Watch_Call) Return(_a0 watch.Interface, _a1 error) *mockEcosystemBackupInterface_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEcosystemBackupInterface_Watch_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (watch.Interface, error)) *mockEcosystemBackupInterface_Watch_Call {
	_c.Call.Return(run)
	return _c
}

// newMockEcosystemBackupInterface creates a new instance of mockEcosystemBackupInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockEcosystemBackupInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockEcosystemBackupInterface {
	mock := &mockEcosystemBackupInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
