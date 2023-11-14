// Code generated by mockery v2.20.0. DO NOT EDIT.

package retention

import (
	context "context"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mock "github.com/stretchr/testify/mock"

	types "k8s.io/apimachinery/pkg/types"

	v1 "k8s.io/client-go/applyconfigurations/core/v1"

	watch "k8s.io/apimachinery/pkg/watch"
)

// mockConfigMapClient is an autogenerated mock type for the configMapClient type
type mockConfigMapClient struct {
	mock.Mock
}

type mockConfigMapClient_Expecter struct {
	mock *mock.Mock
}

func (_m *mockConfigMapClient) EXPECT() *mockConfigMapClient_Expecter {
	return &mockConfigMapClient_Expecter{mock: &_m.Mock}
}

// Apply provides a mock function with given fields: ctx, configMap, opts
func (_m *mockConfigMapClient) Apply(ctx context.Context, configMap *v1.ConfigMapApplyConfiguration, opts metav1.ApplyOptions) (*corev1.ConfigMap, error) {
	ret := _m.Called(ctx, configMap, opts)

	var r0 *corev1.ConfigMap
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *v1.ConfigMapApplyConfiguration, metav1.ApplyOptions) (*corev1.ConfigMap, error)); ok {
		return rf(ctx, configMap, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *v1.ConfigMapApplyConfiguration, metav1.ApplyOptions) *corev1.ConfigMap); ok {
		r0 = rf(ctx, configMap, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.ConfigMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *v1.ConfigMapApplyConfiguration, metav1.ApplyOptions) error); ok {
		r1 = rf(ctx, configMap, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockConfigMapClient_Apply_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Apply'
type mockConfigMapClient_Apply_Call struct {
	*mock.Call
}

// Apply is a helper method to define mock.On call
//   - ctx context.Context
//   - configMap *v1.ConfigMapApplyConfiguration
//   - opts metav1.ApplyOptions
func (_e *mockConfigMapClient_Expecter) Apply(ctx interface{}, configMap interface{}, opts interface{}) *mockConfigMapClient_Apply_Call {
	return &mockConfigMapClient_Apply_Call{Call: _e.mock.On("Apply", ctx, configMap, opts)}
}

func (_c *mockConfigMapClient_Apply_Call) Run(run func(ctx context.Context, configMap *v1.ConfigMapApplyConfiguration, opts metav1.ApplyOptions)) *mockConfigMapClient_Apply_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*v1.ConfigMapApplyConfiguration), args[2].(metav1.ApplyOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_Apply_Call) Return(result *corev1.ConfigMap, err error) *mockConfigMapClient_Apply_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockConfigMapClient_Apply_Call) RunAndReturn(run func(context.Context, *v1.ConfigMapApplyConfiguration, metav1.ApplyOptions) (*corev1.ConfigMap, error)) *mockConfigMapClient_Apply_Call {
	_c.Call.Return(run)
	return _c
}

// Create provides a mock function with given fields: ctx, configMap, opts
func (_m *mockConfigMapClient) Create(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.CreateOptions) (*corev1.ConfigMap, error) {
	ret := _m.Called(ctx, configMap, opts)

	var r0 *corev1.ConfigMap
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.ConfigMap, metav1.CreateOptions) (*corev1.ConfigMap, error)); ok {
		return rf(ctx, configMap, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.ConfigMap, metav1.CreateOptions) *corev1.ConfigMap); ok {
		r0 = rf(ctx, configMap, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.ConfigMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *corev1.ConfigMap, metav1.CreateOptions) error); ok {
		r1 = rf(ctx, configMap, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockConfigMapClient_Create_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Create'
type mockConfigMapClient_Create_Call struct {
	*mock.Call
}

// Create is a helper method to define mock.On call
//   - ctx context.Context
//   - configMap *corev1.ConfigMap
//   - opts metav1.CreateOptions
func (_e *mockConfigMapClient_Expecter) Create(ctx interface{}, configMap interface{}, opts interface{}) *mockConfigMapClient_Create_Call {
	return &mockConfigMapClient_Create_Call{Call: _e.mock.On("Create", ctx, configMap, opts)}
}

func (_c *mockConfigMapClient_Create_Call) Run(run func(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.CreateOptions)) *mockConfigMapClient_Create_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*corev1.ConfigMap), args[2].(metav1.CreateOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_Create_Call) Return(_a0 *corev1.ConfigMap, _a1 error) *mockConfigMapClient_Create_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockConfigMapClient_Create_Call) RunAndReturn(run func(context.Context, *corev1.ConfigMap, metav1.CreateOptions) (*corev1.ConfigMap, error)) *mockConfigMapClient_Create_Call {
	_c.Call.Return(run)
	return _c
}

// Delete provides a mock function with given fields: ctx, name, opts
func (_m *mockConfigMapClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	ret := _m.Called(ctx, name, opts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.DeleteOptions) error); ok {
		r0 = rf(ctx, name, opts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockConfigMapClient_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type mockConfigMapClient_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.DeleteOptions
func (_e *mockConfigMapClient_Expecter) Delete(ctx interface{}, name interface{}, opts interface{}) *mockConfigMapClient_Delete_Call {
	return &mockConfigMapClient_Delete_Call{Call: _e.mock.On("Delete", ctx, name, opts)}
}

func (_c *mockConfigMapClient_Delete_Call) Run(run func(ctx context.Context, name string, opts metav1.DeleteOptions)) *mockConfigMapClient_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.DeleteOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_Delete_Call) Return(_a0 error) *mockConfigMapClient_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockConfigMapClient_Delete_Call) RunAndReturn(run func(context.Context, string, metav1.DeleteOptions) error) *mockConfigMapClient_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteCollection provides a mock function with given fields: ctx, opts, listOpts
func (_m *mockConfigMapClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	ret := _m.Called(ctx, opts, listOpts)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error); ok {
		r0 = rf(ctx, opts, listOpts)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockConfigMapClient_DeleteCollection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteCollection'
type mockConfigMapClient_DeleteCollection_Call struct {
	*mock.Call
}

// DeleteCollection is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.DeleteOptions
//   - listOpts metav1.ListOptions
func (_e *mockConfigMapClient_Expecter) DeleteCollection(ctx interface{}, opts interface{}, listOpts interface{}) *mockConfigMapClient_DeleteCollection_Call {
	return &mockConfigMapClient_DeleteCollection_Call{Call: _e.mock.On("DeleteCollection", ctx, opts, listOpts)}
}

func (_c *mockConfigMapClient_DeleteCollection_Call) Run(run func(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions)) *mockConfigMapClient_DeleteCollection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.DeleteOptions), args[2].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_DeleteCollection_Call) Return(_a0 error) *mockConfigMapClient_DeleteCollection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockConfigMapClient_DeleteCollection_Call) RunAndReturn(run func(context.Context, metav1.DeleteOptions, metav1.ListOptions) error) *mockConfigMapClient_DeleteCollection_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: ctx, name, opts
func (_m *mockConfigMapClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.ConfigMap, error) {
	ret := _m.Called(ctx, name, opts)

	var r0 *corev1.ConfigMap
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) (*corev1.ConfigMap, error)); ok {
		return rf(ctx, name, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, metav1.GetOptions) *corev1.ConfigMap); ok {
		r0 = rf(ctx, name, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.ConfigMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, metav1.GetOptions) error); ok {
		r1 = rf(ctx, name, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockConfigMapClient_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockConfigMapClient_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - opts metav1.GetOptions
func (_e *mockConfigMapClient_Expecter) Get(ctx interface{}, name interface{}, opts interface{}) *mockConfigMapClient_Get_Call {
	return &mockConfigMapClient_Get_Call{Call: _e.mock.On("Get", ctx, name, opts)}
}

func (_c *mockConfigMapClient_Get_Call) Run(run func(ctx context.Context, name string, opts metav1.GetOptions)) *mockConfigMapClient_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(metav1.GetOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_Get_Call) Return(_a0 *corev1.ConfigMap, _a1 error) *mockConfigMapClient_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockConfigMapClient_Get_Call) RunAndReturn(run func(context.Context, string, metav1.GetOptions) (*corev1.ConfigMap, error)) *mockConfigMapClient_Get_Call {
	_c.Call.Return(run)
	return _c
}

// List provides a mock function with given fields: ctx, opts
func (_m *mockConfigMapClient) List(ctx context.Context, opts metav1.ListOptions) (*corev1.ConfigMapList, error) {
	ret := _m.Called(ctx, opts)

	var r0 *corev1.ConfigMapList
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) (*corev1.ConfigMapList, error)); ok {
		return rf(ctx, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, metav1.ListOptions) *corev1.ConfigMapList); ok {
		r0 = rf(ctx, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.ConfigMapList)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, metav1.ListOptions) error); ok {
		r1 = rf(ctx, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockConfigMapClient_List_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'List'
type mockConfigMapClient_List_Call struct {
	*mock.Call
}

// List is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockConfigMapClient_Expecter) List(ctx interface{}, opts interface{}) *mockConfigMapClient_List_Call {
	return &mockConfigMapClient_List_Call{Call: _e.mock.On("List", ctx, opts)}
}

func (_c *mockConfigMapClient_List_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockConfigMapClient_List_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_List_Call) Return(_a0 *corev1.ConfigMapList, _a1 error) *mockConfigMapClient_List_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockConfigMapClient_List_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (*corev1.ConfigMapList, error)) *mockConfigMapClient_List_Call {
	_c.Call.Return(run)
	return _c
}

// Patch provides a mock function with given fields: ctx, name, pt, data, opts, subresources
func (_m *mockConfigMapClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*corev1.ConfigMap, error) {
	_va := make([]interface{}, len(subresources))
	for _i := range subresources {
		_va[_i] = subresources[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, ctx, name, pt, data, opts)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 *corev1.ConfigMap
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*corev1.ConfigMap, error)); ok {
		return rf(ctx, name, pt, data, opts, subresources...)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) *corev1.ConfigMap); ok {
		r0 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.ConfigMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) error); ok {
		r1 = rf(ctx, name, pt, data, opts, subresources...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockConfigMapClient_Patch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Patch'
type mockConfigMapClient_Patch_Call struct {
	*mock.Call
}

// Patch is a helper method to define mock.On call
//   - ctx context.Context
//   - name string
//   - pt types.PatchType
//   - data []byte
//   - opts metav1.PatchOptions
//   - subresources ...string
func (_e *mockConfigMapClient_Expecter) Patch(ctx interface{}, name interface{}, pt interface{}, data interface{}, opts interface{}, subresources ...interface{}) *mockConfigMapClient_Patch_Call {
	return &mockConfigMapClient_Patch_Call{Call: _e.mock.On("Patch",
		append([]interface{}{ctx, name, pt, data, opts}, subresources...)...)}
}

func (_c *mockConfigMapClient_Patch_Call) Run(run func(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string)) *mockConfigMapClient_Patch_Call {
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

func (_c *mockConfigMapClient_Patch_Call) Return(result *corev1.ConfigMap, err error) *mockConfigMapClient_Patch_Call {
	_c.Call.Return(result, err)
	return _c
}

func (_c *mockConfigMapClient_Patch_Call) RunAndReturn(run func(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (*corev1.ConfigMap, error)) *mockConfigMapClient_Patch_Call {
	_c.Call.Return(run)
	return _c
}

// Update provides a mock function with given fields: ctx, configMap, opts
func (_m *mockConfigMapClient) Update(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.UpdateOptions) (*corev1.ConfigMap, error) {
	ret := _m.Called(ctx, configMap, opts)

	var r0 *corev1.ConfigMap
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.ConfigMap, metav1.UpdateOptions) (*corev1.ConfigMap, error)); ok {
		return rf(ctx, configMap, opts)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *corev1.ConfigMap, metav1.UpdateOptions) *corev1.ConfigMap); ok {
		r0 = rf(ctx, configMap, opts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*corev1.ConfigMap)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *corev1.ConfigMap, metav1.UpdateOptions) error); ok {
		r1 = rf(ctx, configMap, opts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockConfigMapClient_Update_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Update'
type mockConfigMapClient_Update_Call struct {
	*mock.Call
}

// Update is a helper method to define mock.On call
//   - ctx context.Context
//   - configMap *corev1.ConfigMap
//   - opts metav1.UpdateOptions
func (_e *mockConfigMapClient_Expecter) Update(ctx interface{}, configMap interface{}, opts interface{}) *mockConfigMapClient_Update_Call {
	return &mockConfigMapClient_Update_Call{Call: _e.mock.On("Update", ctx, configMap, opts)}
}

func (_c *mockConfigMapClient_Update_Call) Run(run func(ctx context.Context, configMap *corev1.ConfigMap, opts metav1.UpdateOptions)) *mockConfigMapClient_Update_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*corev1.ConfigMap), args[2].(metav1.UpdateOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_Update_Call) Return(_a0 *corev1.ConfigMap, _a1 error) *mockConfigMapClient_Update_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockConfigMapClient_Update_Call) RunAndReturn(run func(context.Context, *corev1.ConfigMap, metav1.UpdateOptions) (*corev1.ConfigMap, error)) *mockConfigMapClient_Update_Call {
	_c.Call.Return(run)
	return _c
}

// Watch provides a mock function with given fields: ctx, opts
func (_m *mockConfigMapClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
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

// mockConfigMapClient_Watch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Watch'
type mockConfigMapClient_Watch_Call struct {
	*mock.Call
}

// Watch is a helper method to define mock.On call
//   - ctx context.Context
//   - opts metav1.ListOptions
func (_e *mockConfigMapClient_Expecter) Watch(ctx interface{}, opts interface{}) *mockConfigMapClient_Watch_Call {
	return &mockConfigMapClient_Watch_Call{Call: _e.mock.On("Watch", ctx, opts)}
}

func (_c *mockConfigMapClient_Watch_Call) Run(run func(ctx context.Context, opts metav1.ListOptions)) *mockConfigMapClient_Watch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(metav1.ListOptions))
	})
	return _c
}

func (_c *mockConfigMapClient_Watch_Call) Return(_a0 watch.Interface, _a1 error) *mockConfigMapClient_Watch_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockConfigMapClient_Watch_Call) RunAndReturn(run func(context.Context, metav1.ListOptions) (watch.Interface, error)) *mockConfigMapClient_Watch_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockConfigMapClient interface {
	mock.TestingT
	Cleanup(func())
}

// newMockConfigMapClient creates a new instance of mockConfigMapClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockConfigMapClient(t mockConstructorTestingTnewMockConfigMapClient) *mockConfigMapClient {
	mock := &mockConfigMapClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
