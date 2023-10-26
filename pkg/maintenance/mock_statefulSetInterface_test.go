// Code generated by mockery v2.20.0. DO NOT EDIT.

package maintenance

import (
	appsv1 "k8s.io/api/apps/v1"
	apiautoscalingv1 "k8s.io/api/autoscaling/v1"

	autoscalingv1 "k8s.io/client-go/applyconfigurations/autoscaling/v1"

	context "context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/client-go/applyconfigurations/apps/v1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// mockStatefulSetInterface is an autogenerated mock type for the statefulSetInterface type
type mockStatefulSetInterface struct {
	mock.Mock
}

type mockStatefulSetInterface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockStatefulSetInterface) EXPECT() *mockStatefulSetInterface_Expecter {
	return &mockStatefulSetInterface_Expecter{mock: &_m.Mock}
}

// Apply provides a mock function with given fields: ctx, statefulSet, opts
func (_m *mockStatefulSetInterface) Apply(ctx context.Context, statefulSet *v1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions) (*appsv1.StatefulSet, error) {
	ret := _m.Called(ctx, statefulSet, opts)

	var r0 *appsv1.StatefulSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) (*appsv1.StatefulSet, error)); ok {
		return rf(ctx, statefulSet, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, statefulSet, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, statefulSet, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_Apply_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Apply'
type mockStatefulSetInterface_Apply_Call struct {
	*mock.Call
}

// Apply is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSet *v1.StatefulSetApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockStatefulSetInterface_Expecter) Apply(ctx interface{}, statefulSet interface{}, opts interface{}) *mockStatefulSetInterface_Apply_Call {
	return &mockStatefulSetInterface_Apply_Call{Call: _e.mock.On("Apply", ctx, statefulSet, opts)}
}

func (_c *mockStatefulSetInterface_Apply_Call) Run(run func(ctx context.Context, statefulSet *v1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions)) *mockStatefulSetInterface_Apply_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.StatefulSetApplyConfiguration), args[2].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_Apply_Call) Return(result *appsv1.StatefulSet, err error) *mockStatefulSetInterface_Apply_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockStatefulSetInterface_Apply_Call) RunAndReturn(run func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) (*appsv1.StatefulSet, error)) *mockStatefulSetInterface_Apply_Call {
	_c.Call.Return(run)
	return _c
}

// ApplyScale provides a mock function with given fields: ctx, statefulSetName, scale, opts
func (_m *mockStatefulSetInterface) ApplyScale(ctx context.Context, statefulSetName string, scale *autoscalingv1.ScaleApplyConfiguration, opts metav1.ApplyOptions) (*apiautoscalingv1.Scale, error) {
	ret := _m.Called(ctx, statefulSetName, scale, opts)

	var r0 *apiautoscalingv1.Scale
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *autoscalingv1.ScaleApplyConfiguration, metav1.ApplyOptions) (*apiautoscalingv1.Scale, error)); ok {
		return rf(ctx, statefulSetName, scale, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *autoscalingv1.ScaleApplyConfiguration, metav1.ApplyOptions) *apiautoscalingv1.Scale); ok {
		r0 = rf(ctx, statefulSetName, scale, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*apiautoscalingv1.Scale)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *autoscalingv1.ScaleApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, statefulSetName, scale, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_ApplyScale_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ApplyScale'
type mockStatefulSetInterface_ApplyScale_Call struct {
	*mock.Call
}

// ApplyScale is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSetName string
//   - scale *autoscalingv1.ScaleApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockStatefulSetInterface_Expecter) ApplyScale(ctx interface{}, statefulSetName interface{}, scale interface{}, opts interface{}) *mockStatefulSetInterface_ApplyScale_Call {
	return &mockStatefulSetInterface_ApplyScale_Call{Call: _e.mock.On("ApplyScale", ctx, statefulSetName, scale, opts)}
}

func (_c *mockStatefulSetInterface_ApplyScale_Call) Run(run func(ctx context.Context, statefulSetName string, scale *autoscalingv1.ScaleApplyConfiguration, opts metav1.ApplyOptions)) *mockStatefulSetInterface_ApplyScale_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(*autoscalingv1.ScaleApplyConfiguration), args[3].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_ApplyScale_Call) Return(_a0 *apiautoscalingv1.Scale, _a1 error) *mockStatefulSetInterface_ApplyScale_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_ApplyScale_Call) RunAndReturn(run func(context.Context, string, *autoscalingv1.ScaleApplyConfiguration, metav1.ApplyOptions) (*apiautoscalingv1.Scale, error)) *mockStatefulSetInterface_ApplyScale_Call {
	_c.Call.Return(run)
	return _c
}

// ApplyStatus provides a mock function with given fields: ctx, statefulSet, opts
func (_m *mockStatefulSetInterface) ApplyStatus(ctx context.Context, statefulSet *v1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions) (*appsv1.StatefulSet, error) {
	ret := _m.Called(ctx, statefulSet, opts)

	var r0 *appsv1.StatefulSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) (*appsv1.StatefulSet, error)); ok {
		return rf(ctx, statefulSet, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, statefulSet, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, statefulSet, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_ApplyStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ApplyStatus'
type mockStatefulSetInterface_ApplyStatus_Call struct {
	*mock.Call
}

// ApplyStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSet *v1.StatefulSetApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockStatefulSetInterface_Expecter) ApplyStatus(ctx interface{}, statefulSet interface{}, opts interface{}) *mockStatefulSetInterface_ApplyStatus_Call {
	return &mockStatefulSetInterface_ApplyStatus_Call{Call: _e.mock.On("ApplyStatus", ctx, statefulSet, opts)}
}

func (_c *mockStatefulSetInterface_ApplyStatus_Call) Run(run func(ctx context.Context, statefulSet *v1.StatefulSetApplyConfiguration, opts metav1.ApplyOptions)) *mockStatefulSetInterface_ApplyStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.StatefulSetApplyConfiguration), args[2].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_ApplyStatus_Call) Return(result *appsv1.StatefulSet, err error) *mockStatefulSetInterface_ApplyStatus_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockStatefulSetInterface_ApplyStatus_Call) RunAndReturn(run func(context.Context, *v1.StatefulSetApplyConfiguration, metav1.ApplyOptions) (*appsv1.StatefulSet, error)) *mockStatefulSetInterface_ApplyStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Create provides a mock function with given fields: ctx, statefulSet, opts
func (_m *mockStatefulSetInterface) Create(ctx context.Context, statefulSet *appsv1.StatefulSet, opts metav1.CreateOptions) (*appsv1.StatefulSet, error) {
	ret := _m.Called(ctx, statefulSet, opts)

	var r0 *appsv1.StatefulSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *appsv1.StatefulSet, metav1.CreateOptions) (*appsv1.StatefulSet, error)); ok {
		return rf(ctx, statefulSet, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *appsv1.StatefulSet, metav1.CreateOptions) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, statefulSet, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *appsv1.StatefulSet, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, statefulSet, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type mockStatefulSetInterface_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSet *appsv1.StatefulSet
//   - opts metav1.CreateOptions
func (_e *mockStatefulSetInterface_Expecter) Create(ctx interface{}, statefulSet interface{}, opts interface{}) *mockStatefulSetInterface_Create_Call {
	return &mockStatefulSetInterface_Create_Call{Call: _e.mock.On("Create", ctx, statefulSet, opts)}
}

func (_c *mockStatefulSetInterface_Create_Call) Run(run func(ctx context.Context, statefulSet *appsv1.StatefulSet, opts metav1.CreateOptions)) *mockStatefulSetInterface_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*appsv1.StatefulSet), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_Create_Call) Return(_a0 *appsv1.StatefulSet, _a1 error) *mockStatefulSetInterface_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_Create_Call) RunAndReturn(run func(context.Context, *appsv1.StatefulSet, metav1.CreateOptions) (*appsv1.StatefulSet, error)) *mockStatefulSetInterface_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *mockStatefulSetInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockStatefulSetInterface_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type mockStatefulSetInterface_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.DeleteOptions
func (_e *mockStatefulSetInterface_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *mockStatefulSetInterface_Delete_Call {
	return &mockStatefulSetInterface_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *mockStatefulSetInterface_Delete_Call) Run(run func(ctx context.Context, name string, opts metav1.DeleteOptions)) *mockStatefulSetInterface_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.DeleteOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_Delete_Call) Return(_a0 error) *mockStatefulSetInterface_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockStatefulSetInterface_Delete_Call) RunAndReturn(run func(context.Context, string, metav1.DeleteOptions) error) *mockStatefulSetInterface_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *mockStatefulSetInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockStatefulSetInterface_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type mockStatefulSetInterface_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.DeleteOptions
//   - listOpts metav1.ListOptions
func (_e *mockStatefulSetInterface_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *mockStatefulSetInterface_DeleteCollection_Call {
	return &mockStatefulSetInterface_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *mockStatefulSetInterface_DeleteCollection_Call) Run(run func(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions)) *mockStatefulSetInterface_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.DeleteOptions), args[2].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_DeleteCollection_Call) Return(_a0 error) *mockStatefulSetInterface_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockStatefulSetInterface_DeleteCollection_Call) RunAndReturn(run func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) *mockStatefulSetInterface_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *mockStatefulSetInterface) Get(ctx context.Context, name string, opts metav1.GetOptions) (*appsv1.StatefulSet, error) {
	ret := _m.Called(ctx, name, opts)

	var r0 *appsv1.StatefulSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) (*appsv1.StatefulSet, error)); ok {
		return rf(ctx, name, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockStatefulSetInterface_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.GetOptions
func (_e *mockStatefulSetInterface_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *mockStatefulSetInterface_Get_Call {
	return &mockStatefulSetInterface_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *mockStatefulSetInterface_Get_Call) Run(run func(ctx context.Context, name string, opts metav1.GetOptions)) *mockStatefulSetInterface_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_Get_Call) Return(_a0 *appsv1.StatefulSet, _a1 error) *mockStatefulSetInterface_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_Get_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*appsv1.StatefulSet, error)) *mockStatefulSetInterface_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetScale provides a mock function with given fields: ctx, statefulSetName, options
func (_m *mockStatefulSetInterface) GetScale(ctx context.Context, statefulSetName string, options metav1.GetOptions) (*apiautoscalingv1.Scale, error) {
	ret := _m.Called(ctx, statefulSetName, options)

	var r0 *apiautoscalingv1.Scale
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) (*apiautoscalingv1.Scale, error)); ok {
		return rf(ctx, statefulSetName, options)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *apiautoscalingv1.Scale); ok {
		r0 = rf(ctx, statefulSetName, options)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*apiautoscalingv1.Scale)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, statefulSetName, options)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_GetScale_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetScale'
type mockStatefulSetInterface_GetScale_Call struct {
	*mock.Call
}

// GetScale is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSetName string
//   - options metav1.GetOptions
func (_e *mockStatefulSetInterface_Expecter) GetScale(ctx interface{}, statefulSetName interface{}, options interface{}) *mockStatefulSetInterface_GetScale_Call {
	return &mockStatefulSetInterface_GetScale_Call{Call: _e.mock.On("GetScale", ctx, statefulSetName, options)}
}

func (_c *mockStatefulSetInterface_GetScale_Call) Run(run func(ctx context.Context, statefulSetName string, options metav1.GetOptions)) *mockStatefulSetInterface_GetScale_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_GetScale_Call) Return(_a0 *apiautoscalingv1.Scale, _a1 error) *mockStatefulSetInterface_GetScale_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_GetScale_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*apiautoscalingv1.Scale, error)) *mockStatefulSetInterface_GetScale_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *mockStatefulSetInterface) List(ctx context.Context, opts metav1.ListOptions) (*appsv1.StatefulSetList, error) {
	ret := _m.Called(ctx, opts)

	var r0 *appsv1.StatefulSetList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (*appsv1.StatefulSetList, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) *appsv1.StatefulSetList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSetList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type mockStatefulSetInterface_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockStatefulSetInterface_Expecter) List(ctx interface{}, opts interface{}) *mockStatefulSetInterface_List_Call {
	return &mockStatefulSetInterface_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *mockStatefulSetInterface_List_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockStatefulSetInterface_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_List_Call) Return(_a0 *appsv1.StatefulSetList, _a1 error) *mockStatefulSetInterface_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_List_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (*appsv1.StatefulSetList, error)) *mockStatefulSetInterface_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *mockStatefulSetInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*appsv1.StatefulSet, error) {
	_va := make([]interface{}, len(subresources))
	for _i := range subresources {
		_va[_i] = subresources[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name, pt, data, opts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *appsv1.StatefulSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*appsv1.StatefulSet, error)); ok {
		return rf(ctx, name, pt, data, opts, subresources...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type mockStatefulSetInterface_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts metav1.PatchOptions
//   - subresources ...string
func (_e *mockStatefulSetInterface_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *mockStatefulSetInterface_Patch_Call {
	return &mockStatefulSetInterface_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *mockStatefulSetInterface_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string)) *mockStatefulSetInterface_Patch_Call {
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

func (_c *mockStatefulSetInterface_Patch_Call) Return(result *appsv1.StatefulSet, err error) *mockStatefulSetInterface_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockStatefulSetInterface_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*appsv1.StatefulSet, error)) *mockStatefulSetInterface_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, statefulSet, opts
func (_m *mockStatefulSetInterface) Update(ctx context.Context, statefulSet *appsv1.StatefulSet, opts metav1.UpdateOptions) (*appsv1.StatefulSet, error) {
	ret := _m.Called(ctx, statefulSet, opts)

	var r0 *appsv1.StatefulSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) (*appsv1.StatefulSet, error)); ok {
		return rf(ctx, statefulSet, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, statefulSet, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, statefulSet, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockStatefulSetInterface_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSet *appsv1.StatefulSet
//   - opts metav1.UpdateOptions
func (_e *mockStatefulSetInterface_Expecter) Update(ctx interface{}, statefulSet interface{}, opts interface{}) *mockStatefulSetInterface_Update_Call {
	return &mockStatefulSetInterface_Update_Call{Call: _e.mock.On("Update", ctx, statefulSet, opts)}
}

func (_c *mockStatefulSetInterface_Update_Call) Run(run func(ctx context.Context, statefulSet *appsv1.StatefulSet, opts metav1.UpdateOptions)) *mockStatefulSetInterface_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*appsv1.StatefulSet), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_Update_Call) Return(_a0 *appsv1.StatefulSet, _a1 error) *mockStatefulSetInterface_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_Update_Call) RunAndReturn(run func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) (*appsv1.StatefulSet, error)) *mockStatefulSetInterface_Update_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateScale provides a mock function with given fields: ctx, statefulSetName, scale, opts
func (_m *mockStatefulSetInterface) UpdateScale(ctx context.Context, statefulSetName string, scale *apiautoscalingv1.Scale, opts metav1.UpdateOptions) (*apiautoscalingv1.Scale, error) {
	ret := _m.Called(ctx, statefulSetName, scale, opts)

	var r0 *apiautoscalingv1.Scale
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *apiautoscalingv1.Scale, metav1.UpdateOptions) (*apiautoscalingv1.Scale, error)); ok {
		return rf(ctx, statefulSetName, scale, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *apiautoscalingv1.Scale, metav1.UpdateOptions) *apiautoscalingv1.Scale); ok {
		r0 = rf(ctx, statefulSetName, scale, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*apiautoscalingv1.Scale)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *apiautoscalingv1.Scale, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, statefulSetName, scale, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_UpdateScale_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateScale'
type mockStatefulSetInterface_UpdateScale_Call struct {
	*mock.Call
}

// UpdateScale is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSetName string
//   - scale *apiautoscalingv1.Scale
//   - opts metav1.UpdateOptions
func (_e *mockStatefulSetInterface_Expecter) UpdateScale(ctx interface{}, statefulSetName interface{}, scale interface{}, opts interface{}) *mockStatefulSetInterface_UpdateScale_Call {
	return &mockStatefulSetInterface_UpdateScale_Call{Call: _e.mock.On("UpdateScale", ctx, statefulSetName, scale, opts)}
}

func (_c *mockStatefulSetInterface_UpdateScale_Call) Run(run func(ctx context.Context, statefulSetName string, scale *apiautoscalingv1.Scale, opts metav1.UpdateOptions)) *mockStatefulSetInterface_UpdateScale_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(*apiautoscalingv1.Scale), args[3].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_UpdateScale_Call) Return(_a0 *apiautoscalingv1.Scale, _a1 error) *mockStatefulSetInterface_UpdateScale_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_UpdateScale_Call) RunAndReturn(run func(context.Context, string, *apiautoscalingv1.Scale, metav1.UpdateOptions) (*apiautoscalingv1.Scale, error)) *mockStatefulSetInterface_UpdateScale_Call {
	_c.Call.Return(run)
	return _c
}

// UpdateStatus provides a mock function with given fields: ctx, statefulSet, opts
func (_m *mockStatefulSetInterface) UpdateStatus(ctx context.Context, statefulSet *appsv1.StatefulSet, opts metav1.UpdateOptions) (*appsv1.StatefulSet, error) {
	ret := _m.Called(ctx, statefulSet, opts)

	var r0 *appsv1.StatefulSet
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) (*appsv1.StatefulSet, error)); ok {
		return rf(ctx, statefulSet, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) *appsv1.StatefulSet); ok {
		r0 = rf(ctx, statefulSet, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*appsv1.StatefulSet)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, statefulSet, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockStatefulSetInterface_UpdateStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'UpdateStatus'
type mockStatefulSetInterface_UpdateStatus_Call struct {
	*mock.Call
}

// UpdateStatus is a helper method to define mock.On call
//   - ctx context.Context
//   - statefulSet *appsv1.StatefulSet
//   - opts metav1.UpdateOptions
func (_e *mockStatefulSetInterface_Expecter) UpdateStatus(ctx interface{}, statefulSet interface{}, opts interface{}) *mockStatefulSetInterface_UpdateStatus_Call {
	return &mockStatefulSetInterface_UpdateStatus_Call{Call: _e.mock.On("UpdateStatus", ctx, statefulSet, opts)}
}

func (_c *mockStatefulSetInterface_UpdateStatus_Call) Run(run func(ctx context.Context, statefulSet *appsv1.StatefulSet, opts metav1.UpdateOptions)) *mockStatefulSetInterface_UpdateStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*appsv1.StatefulSet), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_UpdateStatus_Call) Return(_a0 *appsv1.StatefulSet, _a1 error) *mockStatefulSetInterface_UpdateStatus_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_UpdateStatus_Call) RunAndReturn(run func(context.Context, *appsv1.StatefulSet, metav1.UpdateOptions) (*appsv1.StatefulSet, error)) *mockStatefulSetInterface_UpdateStatus_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *mockStatefulSetInterface) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	ret := _m.Called(ctx, opts)

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

// mockStatefulSetInterface_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type mockStatefulSetInterface_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockStatefulSetInterface_Expecter) Watch(ctx interface{}, opts interface{}) *mockStatefulSetInterface_Watch_Call {
	return &mockStatefulSetInterface_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *mockStatefulSetInterface_Watch_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockStatefulSetInterface_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockStatefulSetInterface_Watch_Call) Return(_a0 watch.Interface, _a1 error) *mockStatefulSetInterface_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockStatefulSetInterface_Watch_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (watch.Interface, error)) *mockStatefulSetInterface_Watch_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockStatefulSetInterface interface {
	mock.TestingT
	Cleanup(func())
}

// newMockStatefulSetInterface creates a new instance of mockStatefulSetInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockStatefulSetInterface(t mockConstructorTestingTnewMockStatefulSetInterface) *mockStatefulSetInterface {
	mock := &mockStatefulSetInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
