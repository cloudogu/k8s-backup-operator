// Code generated by mockery v2.42.1. DO NOT EDIT.

package additionalimages

import (
	context "context"

	batchv1 "k8s.io/api/batch/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/client-go/applyconfigurations/batch/v1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// mockCronJobInterface is an autogenerated mock type for the cronJobInterface type
type mockCronJobInterface struct {
	mock.Mock
}

type mockCronJobInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockCronJobInterface) EXPECT() *mockCronJobInterface_Expecter {
	return &mockCronJobInterface_Expecter{mock: &_m.Mock}
}

// Apply provides a mock function with given fields: ctx, cronJob, opts
func (_m *mockCronJobInterface) Apply(ctx context.Context, cronJob *v1.CronJobApplyConfiguration, opts metav1.ApplyOptions) (*batchv1.CronJob, error) {
	ret := _m.Called(ctx, cronJob, opts)

	if len(ret) == 0 {
		panic("no return value specified for Apply")
	}

	var r0 *batchv1.CronJob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) (*batchv1.CronJob, error)); ok {
		return rf(ctx, cronJob, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) *batchv1.CronJob); ok {
		r0 = rf(ctx, cronJob, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, cronJob, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_Apply_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Apply'
type mockCronJobInterface_Apply_Call struct {
	*mock.Call
}

// Apply is a helper method to define mock.On call
//   - ctx context.Context
//   - cronJob *v1.CronJobApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockCronJobInterface_Expecter) Apply(ctx interface{}, cronJob interface{}, opts interface{}) *mockCronJobInterface_Apply_Call {
	return &mockCronJobInterface_Apply_Call{Call: _e.mock.On("Apply", ctx, cronJob, opts)}
}

func (_c *mockCronJobInterface_Apply_Call) Run(run func(ctx context.Context, cronJob *v1.CronJobApplyConfiguration, opts metav1.ApplyOptions)) *mockCronJobInterface_Apply_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.CronJobApplyConfiguration), args[2].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_Apply_Call) Return(result *batchv1.CronJob, err error) *mockCronJobInterface_Apply_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockCronJobInterface_Apply_Call) RunAndReturn(run func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) (*batchv1.CronJob, error)) *mockCronJobInterface_Apply_Call {
	_c.Call.Return(run)
	return _c
}

// ApplyStatus provides a mock function with given fields: ctx, cronJob, opts
func (_m *mockCronJobInterface) ApplyStatus(ctx context.Context, cronJob *v1.CronJobApplyConfiguration, opts metav1.ApplyOptions) (*batchv1.CronJob, error) {
	ret := _m.Called(ctx, cronJob, opts)

	if len(ret) == 0 {
		panic("no return value specified for ApplyStatus")
	}

	var r0 *batchv1.CronJob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) (*batchv1.CronJob, error)); ok {
		return rf(ctx, cronJob, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) *batchv1.CronJob); ok {
		r0 = rf(ctx, cronJob, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, cronJob, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_ApplyStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ApplyStatus'
type mockCronJobInterface_ApplyStatus_Call struct {
	*mock.Call
}

// ApplyStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - cronJob *v1.CronJobApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockCronJobInterface_Expecter) ApplyStatus(ctx interface{}, cronJob interface{}, opts interface{}) *mockCronJobInterface_ApplyStatus_Call {
	return &mockCronJobInterface_ApplyStatus_Call{Call: _e.mock.On("ApplyStatus", ctx, cronJob, opts)}
}

func (_c *mockCronJobInterface_ApplyStatus_Call) Run(run func(ctx context.Context, cronJob *v1.CronJobApplyConfiguration, opts metav1.ApplyOptions)) *mockCronJobInterface_ApplyStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.CronJobApplyConfiguration), args[2].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_ApplyStatus_Call) Return(result *batchv1.CronJob, err error) *mockCronJobInterface_ApplyStatus_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockCronJobInterface_ApplyStatus_Call) RunAndReturn(run func(context.Context, *v1.CronJobApplyConfiguration, metav1.ApplyOptions) (*batchv1.CronJob, error)) *mockCronJobInterface_ApplyStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Create provides a mock function with given fields: ctx, cronJob, opts
func (_m *mockCronJobInterface) Create(ctx context.Context, cronJob *batchv1.CronJob, opts metav1.CreateOptions) (*batchv1.CronJob, error) {
	ret := _m.Called(ctx, cronJob, opts)

	if len(ret) == 0 {
		panic("no return value specified for Create")
	}

	var r0 *batchv1.CronJob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *batchv1.CronJob, metav1.CreateOptions) (*batchv1.CronJob, error)); ok {
		return rf(ctx, cronJob, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *batchv1.CronJob, metav1.CreateOptions) *batchv1.CronJob); ok {
		r0 = rf(ctx, cronJob, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *batchv1.CronJob, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, cronJob, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type mockCronJobInterface_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - cronJob *batchv1.CronJob
//   - opts metav1.CreateOptions
func (_e *mockCronJobInterface_Expecter) Create(ctx interface{}, cronJob interface{}, opts interface{}) *mockCronJobInterface_Create_Call {
	return &mockCronJobInterface_Create_Call{Call: _e.mock.On("Create", ctx, cronJob, opts)}
}

func (_c *mockCronJobInterface_Create_Call) Run(run func(ctx context.Context, cronJob *batchv1.CronJob, opts metav1.CreateOptions)) *mockCronJobInterface_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*batchv1.CronJob), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_Create_Call) Return(_a0 *batchv1.CronJob, _a1 error) *mockCronJobInterface_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCronJobInterface_Create_Call) RunAndReturn(run func(context.Context, *batchv1.CronJob, metav1.CreateOptions) (*batchv1.CronJob, error)) *mockCronJobInterface_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *mockCronJobInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
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

// mockCronJobInterface_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type mockCronJobInterface_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.DeleteOptions
func (_e *mockCronJobInterface_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *mockCronJobInterface_Delete_Call {
	return &mockCronJobInterface_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *mockCronJobInterface_Delete_Call) Run(run func(ctx context.Context, name string, opts metav1.DeleteOptions)) *mockCronJobInterface_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.DeleteOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_Delete_Call) Return(_a0 error) *mockCronJobInterface_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockCronJobInterface_Delete_Call) RunAndReturn(run func(context.Context, string, metav1.DeleteOptions) error) *mockCronJobInterface_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *mockCronJobInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
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

// mockCronJobInterface_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type mockCronJobInterface_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.DeleteOptions
//   - listOpts metav1.ListOptions
func (_e *mockCronJobInterface_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *mockCronJobInterface_DeleteCollection_Call {
	return &mockCronJobInterface_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *mockCronJobInterface_DeleteCollection_Call) Run(run func(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions)) *mockCronJobInterface_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.DeleteOptions), args[2].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_DeleteCollection_Call) Return(_a0 error) *mockCronJobInterface_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockCronJobInterface_DeleteCollection_Call) RunAndReturn(run func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) *mockCronJobInterface_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *mockCronJobInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*batchv1.CronJob, error) {
	ret := _m.Called(ctx, name, opts)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 *batchv1.CronJob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) (*batchv1.CronJob, error)); ok {
		return rf(ctx, name, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *batchv1.CronJob); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockCronJobInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.GetOptions
func (_e *mockCronJobInterface_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *mockCronJobInterface_Get_Call {
	return &mockCronJobInterface_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *mockCronJobInterface_Get_Call) Run(run func(ctx context.Context, name string, opts metav1.GetOptions)) *mockCronJobInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_Get_Call) Return(_a0 *batchv1.CronJob, _a1 error) *mockCronJobInterface_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCronJobInterface_Get_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*batchv1.CronJob, error)) *mockCronJobInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *mockCronJobInterface) List(ctx context.Context, opts metav1.ListOptions) (*batchv1.CronJobList, error) {
	ret := _m.Called(ctx, opts)

	if len(ret) == 0 {
		panic("no return value specified for List")
	}

	var r0 *batchv1.CronJobList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (*batchv1.CronJobList, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) *batchv1.CronJobList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJobList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type mockCronJobInterface_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockCronJobInterface_Expecter) List(ctx interface{}, opts interface{}) *mockCronJobInterface_List_Call {
	return &mockCronJobInterface_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *mockCronJobInterface_List_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockCronJobInterface_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_List_Call) Return(_a0 *batchv1.CronJobList, _a1 error) *mockCronJobInterface_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCronJobInterface_List_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (*batchv1.CronJobList, error)) *mockCronJobInterface_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *mockCronJobInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*batchv1.CronJob, error) {
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

	var r0 *batchv1.CronJob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*batchv1.CronJob, error)); ok {
		return rf(ctx, name, pt, data, opts, subresources...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) *batchv1.CronJob); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type mockCronJobInterface_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts metav1.PatchOptions
//   - subresources ...string
func (_e *mockCronJobInterface_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *mockCronJobInterface_Patch_Call {
	return &mockCronJobInterface_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *mockCronJobInterface_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string)) *mockCronJobInterface_Patch_Call {
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

func (_c *mockCronJobInterface_Patch_Call) Return(result *batchv1.CronJob, err error) *mockCronJobInterface_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockCronJobInterface_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*batchv1.CronJob, error)) *mockCronJobInterface_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, cronJob, opts
func (_m *mockCronJobInterface) Update(ctx context.Context, cronJob *batchv1.CronJob, opts metav1.UpdateOptions) (*batchv1.CronJob, error) {
	ret := _m.Called(ctx, cronJob, opts)

	if len(ret) == 0 {
		panic("no return value specified for Update")
	}

	var r0 *batchv1.CronJob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) (*batchv1.CronJob, error)); ok {
		return rf(ctx, cronJob, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) *batchv1.CronJob); ok {
		r0 = rf(ctx, cronJob, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, cronJob, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockCronJobInterface_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - cronJob *batchv1.CronJob
//   - opts metav1.UpdateOptions
func (_e *mockCronJobInterface_Expecter) Update(ctx interface{}, cronJob interface{}, opts interface{}) *mockCronJobInterface_Update_Call {
	return &mockCronJobInterface_Update_Call{Call: _e.mock.On("Update", ctx, cronJob, opts)}
}

func (_c *mockCronJobInterface_Update_Call) Run(run func(ctx context.Context, cronJob *batchv1.CronJob, opts metav1.UpdateOptions)) *mockCronJobInterface_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*batchv1.CronJob), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_Update_Call) Return(_a0 *batchv1.CronJob, _a1 error) *mockCronJobInterface_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCronJobInterface_Update_Call) RunAndReturn(run func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) (*batchv1.CronJob, error)) *mockCronJobInterface_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, cronJob, opts
func (_m *mockCronJobInterface) UpdateStatus(ctx context.Context, cronJob *batchv1.CronJob, opts metav1.UpdateOptions) (*batchv1.CronJob, error) {
	ret := _m.Called(ctx, cronJob, opts)

	if len(ret) == 0 {
		panic("no return value specified for UpdateStatus")
	}

	var r0 *batchv1.CronJob
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) (*batchv1.CronJob, error)); ok {
		return rf(ctx, cronJob, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) *batchv1.CronJob); ok {
		r0 = rf(ctx, cronJob, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*batchv1.CronJob)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, cronJob, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockCronJobInterface_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type mockCronJobInterface_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - cronJob *batchv1.CronJob
//   - opts metav1.UpdateOptions
func (_e *mockCronJobInterface_Expecter) UpdateStatus(ctx interface{}, cronJob interface{}, opts interface{}) *mockCronJobInterface_UpdateStatus_Call {
	return &mockCronJobInterface_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, cronJob, opts)}
}

func (_c *mockCronJobInterface_UpdateStatus_Call) Run(run func(ctx context.Context, cronJob *batchv1.CronJob, opts metav1.UpdateOptions)) *mockCronJobInterface_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*batchv1.CronJob), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_UpdateStatus_Call) Return(_a0 *batchv1.CronJob, _a1 error) *mockCronJobInterface_UpdateStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCronJobInterface_UpdateStatus_Call) RunAndReturn(run func(context.Context, *batchv1.CronJob, metav1.UpdateOptions) (*batchv1.CronJob, error)) *mockCronJobInterface_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *mockCronJobInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
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

// mockCronJobInterface_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type mockCronJobInterface_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockCronJobInterface_Expecter) Watch(ctx interface{}, opts interface{}) *mockCronJobInterface_Watch_Call {
	return &mockCronJobInterface_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *mockCronJobInterface_Watch_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockCronJobInterface_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockCronJobInterface_Watch_Call) Return(_a0 watch.Interface, _a1 error) *mockCronJobInterface_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockCronJobInterface_Watch_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (watch.Interface, error)) *mockCronJobInterface_Watch_Call {
	_c.Call.Return(run)
	return _c
}

// newMockCronJobInterface creates a new instance of mockCronJobInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockCronJobInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockCronJobInterface {
	mock := &mockCronJobInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
