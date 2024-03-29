// Code generated by mockery v2.20.0. DO NOT EDIT.

package main

import mock "github.com/stretchr/testify/mock"

// mockEtcdContext is an autogenerated mock type for the etcdContext type
type mockEtcdContext struct {
	mock.Mock
}

type mockEtcdContext_Expecter struct {
	mock *mock.Mock
}

func (_m *mockEtcdContext) EXPECT() *mockEtcdContext_Expecter {
	return &mockEtcdContext_Expecter{mock: &_m.Mock}
}

// Delete provides a mock function with given fields: key
func (_m *mockEtcdContext) Delete(key string) error {
	ret := _m.Called(key)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockEtcdContext_Delete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Delete'
type mockEtcdContext_Delete_Call struct {
	*mock.Call
}

// Delete is a helper method to define mock.On call
//   - key string
func (_e *mockEtcdContext_Expecter) Delete(key interface{}) *mockEtcdContext_Delete_Call {
	return &mockEtcdContext_Delete_Call{Call: _e.mock.On("Delete", key)}
}

func (_c *mockEtcdContext_Delete_Call) Run(run func(key string)) *mockEtcdContext_Delete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockEtcdContext_Delete_Call) Return(_a0 error) *mockEtcdContext_Delete_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEtcdContext_Delete_Call) RunAndReturn(run func(string) error) *mockEtcdContext_Delete_Call {
	_c.Call.Return(run)
	return _c
}

// DeleteRecursive provides a mock function with given fields: key
func (_m *mockEtcdContext) DeleteRecursive(key string) error {
	ret := _m.Called(key)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockEtcdContext_DeleteRecursive_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeleteRecursive'
type mockEtcdContext_DeleteRecursive_Call struct {
	*mock.Call
}

// DeleteRecursive is a helper method to define mock.On call
//   - key string
func (_e *mockEtcdContext_Expecter) DeleteRecursive(key interface{}) *mockEtcdContext_DeleteRecursive_Call {
	return &mockEtcdContext_DeleteRecursive_Call{Call: _e.mock.On("DeleteRecursive", key)}
}

func (_c *mockEtcdContext_DeleteRecursive_Call) Run(run func(key string)) *mockEtcdContext_DeleteRecursive_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockEtcdContext_DeleteRecursive_Call) Return(_a0 error) *mockEtcdContext_DeleteRecursive_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEtcdContext_DeleteRecursive_Call) RunAndReturn(run func(string) error) *mockEtcdContext_DeleteRecursive_Call {
	_c.Call.Return(run)
	return _c
}

// Exists provides a mock function with given fields: key
func (_m *mockEtcdContext) Exists(key string) (bool, error) {
	ret := _m.Called(key)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (bool, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEtcdContext_Exists_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Exists'
type mockEtcdContext_Exists_Call struct {
	*mock.Call
}

// Exists is a helper method to define mock.On call
//   - key string
func (_e *mockEtcdContext_Expecter) Exists(key interface{}) *mockEtcdContext_Exists_Call {
	return &mockEtcdContext_Exists_Call{Call: _e.mock.On("Exists", key)}
}

func (_c *mockEtcdContext_Exists_Call) Run(run func(key string)) *mockEtcdContext_Exists_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockEtcdContext_Exists_Call) Return(_a0 bool, _a1 error) *mockEtcdContext_Exists_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEtcdContext_Exists_Call) RunAndReturn(run func(string) (bool, error)) *mockEtcdContext_Exists_Call {
	_c.Call.Return(run)
	return _c
}

// Get provides a mock function with given fields: key
func (_m *mockEtcdContext) Get(key string) (string, error) {
	ret := _m.Called(key)

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEtcdContext_Get_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Get'
type mockEtcdContext_Get_Call struct {
	*mock.Call
}

// Get is a helper method to define mock.On call
//   - key string
func (_e *mockEtcdContext_Expecter) Get(key interface{}) *mockEtcdContext_Get_Call {
	return &mockEtcdContext_Get_Call{Call: _e.mock.On("Get", key)}
}

func (_c *mockEtcdContext_Get_Call) Run(run func(key string)) *mockEtcdContext_Get_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockEtcdContext_Get_Call) Return(_a0 string, _a1 error) *mockEtcdContext_Get_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEtcdContext_Get_Call) RunAndReturn(run func(string) (string, error)) *mockEtcdContext_Get_Call {
	_c.Call.Return(run)
	return _c
}

// GetAll provides a mock function with given fields:
func (_m *mockEtcdContext) GetAll() (map[string]string, error) {
	ret := _m.Called()

	var r0 map[string]string
	var r1 error
	if rf, ok := ret.Get(0).(func() (map[string]string, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() map[string]string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// mockEtcdContext_GetAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAll'
type mockEtcdContext_GetAll_Call struct {
	*mock.Call
}

// GetAll is a helper method to define mock.On call
func (_e *mockEtcdContext_Expecter) GetAll() *mockEtcdContext_GetAll_Call {
	return &mockEtcdContext_GetAll_Call{Call: _e.mock.On("GetAll")}
}

func (_c *mockEtcdContext_GetAll_Call) Run(run func()) *mockEtcdContext_GetAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockEtcdContext_GetAll_Call) Return(_a0 map[string]string, _a1 error) *mockEtcdContext_GetAll_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *mockEtcdContext_GetAll_Call) RunAndReturn(run func() (map[string]string, error)) *mockEtcdContext_GetAll_Call {
	_c.Call.Return(run)
	return _c
}

// GetOrFalse provides a mock function with given fields: key
func (_m *mockEtcdContext) GetOrFalse(key string) (bool, string, error) {
	ret := _m.Called(key)

	var r0 bool
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(string) (bool, string, error)); ok {
		return rf(key)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(key)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) string); ok {
		r1 = rf(key)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(string) error); ok {
		r2 = rf(key)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// mockEtcdContext_GetOrFalse_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetOrFalse'
type mockEtcdContext_GetOrFalse_Call struct {
	*mock.Call
}

// GetOrFalse is a helper method to define mock.On call
//   - key string
func (_e *mockEtcdContext_Expecter) GetOrFalse(key interface{}) *mockEtcdContext_GetOrFalse_Call {
	return &mockEtcdContext_GetOrFalse_Call{Call: _e.mock.On("GetOrFalse", key)}
}

func (_c *mockEtcdContext_GetOrFalse_Call) Run(run func(key string)) *mockEtcdContext_GetOrFalse_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockEtcdContext_GetOrFalse_Call) Return(_a0 bool, _a1 string, _a2 error) *mockEtcdContext_GetOrFalse_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *mockEtcdContext_GetOrFalse_Call) RunAndReturn(run func(string) (bool, string, error)) *mockEtcdContext_GetOrFalse_Call {
	_c.Call.Return(run)
	return _c
}

// Refresh provides a mock function with given fields: key, timeToLiveInSeconds
func (_m *mockEtcdContext) Refresh(key string, timeToLiveInSeconds int) error {
	ret := _m.Called(key, timeToLiveInSeconds)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int) error); ok {
		r0 = rf(key, timeToLiveInSeconds)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockEtcdContext_Refresh_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Refresh'
type mockEtcdContext_Refresh_Call struct {
	*mock.Call
}

// Refresh is a helper method to define mock.On call
//   - key string
//   - timeToLiveInSeconds int
func (_e *mockEtcdContext_Expecter) Refresh(key interface{}, timeToLiveInSeconds interface{}) *mockEtcdContext_Refresh_Call {
	return &mockEtcdContext_Refresh_Call{Call: _e.mock.On("Refresh", key, timeToLiveInSeconds)}
}

func (_c *mockEtcdContext_Refresh_Call) Run(run func(key string, timeToLiveInSeconds int)) *mockEtcdContext_Refresh_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(int))
	})
	return _c
}

func (_c *mockEtcdContext_Refresh_Call) Return(_a0 error) *mockEtcdContext_Refresh_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEtcdContext_Refresh_Call) RunAndReturn(run func(string, int) error) *mockEtcdContext_Refresh_Call {
	_c.Call.Return(run)
	return _c
}

// RemoveAll provides a mock function with given fields:
func (_m *mockEtcdContext) RemoveAll() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockEtcdContext_RemoveAll_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RemoveAll'
type mockEtcdContext_RemoveAll_Call struct {
	*mock.Call
}

// RemoveAll is a helper method to define mock.On call
func (_e *mockEtcdContext_Expecter) RemoveAll() *mockEtcdContext_RemoveAll_Call {
	return &mockEtcdContext_RemoveAll_Call{Call: _e.mock.On("RemoveAll")}
}

func (_c *mockEtcdContext_RemoveAll_Call) Run(run func()) *mockEtcdContext_RemoveAll_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockEtcdContext_RemoveAll_Call) Return(_a0 error) *mockEtcdContext_RemoveAll_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEtcdContext_RemoveAll_Call) RunAndReturn(run func() error) *mockEtcdContext_RemoveAll_Call {
	_c.Call.Return(run)
	return _c
}

// Set provides a mock function with given fields: key, value
func (_m *mockEtcdContext) Set(key string, value string) error {
	ret := _m.Called(key, value)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(key, value)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockEtcdContext_Set_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Set'
type mockEtcdContext_Set_Call struct {
	*mock.Call
}

// Set is a helper method to define mock.On call
//   - key string
//   - value string
func (_e *mockEtcdContext_Expecter) Set(key interface{}, value interface{}) *mockEtcdContext_Set_Call {
	return &mockEtcdContext_Set_Call{Call: _e.mock.On("Set", key, value)}
}

func (_c *mockEtcdContext_Set_Call) Run(run func(key string, value string)) *mockEtcdContext_Set_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string))
	})
	return _c
}

func (_c *mockEtcdContext_Set_Call) Return(_a0 error) *mockEtcdContext_Set_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEtcdContext_Set_Call) RunAndReturn(run func(string, string) error) *mockEtcdContext_Set_Call {
	_c.Call.Return(run)
	return _c
}

// SetWithLifetime provides a mock function with given fields: key, value, timeToLiveInSeconds
func (_m *mockEtcdContext) SetWithLifetime(key string, value string, timeToLiveInSeconds int) error {
	ret := _m.Called(key, value, timeToLiveInSeconds)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, int) error); ok {
		r0 = rf(key, value, timeToLiveInSeconds)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// mockEtcdContext_SetWithLifetime_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetWithLifetime'
type mockEtcdContext_SetWithLifetime_Call struct {
	*mock.Call
}

// SetWithLifetime is a helper method to define mock.On call
//   - key string
//   - value string
//   - timeToLiveInSeconds int
func (_e *mockEtcdContext_Expecter) SetWithLifetime(key interface{}, value interface{}, timeToLiveInSeconds interface{}) *mockEtcdContext_SetWithLifetime_Call {
	return &mockEtcdContext_SetWithLifetime_Call{Call: _e.mock.On("SetWithLifetime", key, value, timeToLiveInSeconds)}
}

func (_c *mockEtcdContext_SetWithLifetime_Call) Run(run func(key string, value string, timeToLiveInSeconds int)) *mockEtcdContext_SetWithLifetime_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string), args[2].(int))
	})
	return _c
}

func (_c *mockEtcdContext_SetWithLifetime_Call) Return(_a0 error) *mockEtcdContext_SetWithLifetime_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockEtcdContext_SetWithLifetime_Call) RunAndReturn(run func(string, string, int) error) *mockEtcdContext_SetWithLifetime_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockEtcdContext interface {
	mock.TestingT
	Cleanup(func())
}

// newMockEtcdContext creates a new instance of mockEtcdContext. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockEtcdContext(t mockConstructorTestingTnewMockEtcdContext) *mockEtcdContext {
	mock := &mockEtcdContext{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
