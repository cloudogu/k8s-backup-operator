// Code generated by mockery v2.20.0. DO NOT EDIT.

package v1

import (
	mock "github.com/stretchr/testify/mock"
	runtime "k8s.io/apimachinery/pkg/runtime"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

// MockRequeuableObject is an autogenerated mock type for the RequeuableObject type
type MockRequeuableObject struct {
	mock.Mock
}

type MockRequeuableObject_Expecter struct {
	mock *mock.Mock
}

func (_m *MockRequeuableObject) EXPECT() *MockRequeuableObject_Expecter {
	return &MockRequeuableObject_Expecter{mock: &_m.Mock}
}

// DeepCopyObject provides a mock function with given fields:
func (_m *MockRequeuableObject) DeepCopyObject() runtime.Object {
	ret := _m.Called()

	var r0 runtime.Object
	if rf, ok := ret.Get(0).(func() runtime.Object); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(runtime.Object)
		}
	}

	return r0
}

// MockRequeuableObject_DeepCopyObject_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DeepCopyObject'
type MockRequeuableObject_DeepCopyObject_Call struct {
	*mock.Call
}

// DeepCopyObject is a helper method to define mock.On call
func (_e *MockRequeuableObject_Expecter) DeepCopyObject() *MockRequeuableObject_DeepCopyObject_Call {
	return &MockRequeuableObject_DeepCopyObject_Call{Call: _e.mock.On("DeepCopyObject")}
}

func (_c *MockRequeuableObject_DeepCopyObject_Call) Run(run func()) *MockRequeuableObject_DeepCopyObject_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRequeuableObject_DeepCopyObject_Call) Return(_a0 runtime.Object) *MockRequeuableObject_DeepCopyObject_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRequeuableObject_DeepCopyObject_Call) RunAndReturn(run func() runtime.Object) *MockRequeuableObject_DeepCopyObject_Call {
	_c.Call.Return(run)
	return _c
}

// GetName provides a mock function with given fields:
func (_m *MockRequeuableObject) GetName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// MockRequeuableObject_GetName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetName'
type MockRequeuableObject_GetName_Call struct {
	*mock.Call
}

// GetName is a helper method to define mock.On call
func (_e *MockRequeuableObject_Expecter) GetName() *MockRequeuableObject_GetName_Call {
	return &MockRequeuableObject_GetName_Call{Call: _e.mock.On("GetName")}
}

func (_c *MockRequeuableObject_GetName_Call) Run(run func()) *MockRequeuableObject_GetName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRequeuableObject_GetName_Call) Return(_a0 string) *MockRequeuableObject_GetName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRequeuableObject_GetName_Call) RunAndReturn(run func() string) *MockRequeuableObject_GetName_Call {
	_c.Call.Return(run)
	return _c
}

// GetObjectKind provides a mock function with given fields:
func (_m *MockRequeuableObject) GetObjectKind() schema.ObjectKind {
	ret := _m.Called()

	var r0 schema.ObjectKind
	if rf, ok := ret.Get(0).(func() schema.ObjectKind); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(schema.ObjectKind)
		}
	}

	return r0
}

// MockRequeuableObject_GetObjectKind_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetObjectKind'
type MockRequeuableObject_GetObjectKind_Call struct {
	*mock.Call
}

// GetObjectKind is a helper method to define mock.On call
func (_e *MockRequeuableObject_Expecter) GetObjectKind() *MockRequeuableObject_GetObjectKind_Call {
	return &MockRequeuableObject_GetObjectKind_Call{Call: _e.mock.On("GetObjectKind")}
}

func (_c *MockRequeuableObject_GetObjectKind_Call) Run(run func()) *MockRequeuableObject_GetObjectKind_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRequeuableObject_GetObjectKind_Call) Return(_a0 schema.ObjectKind) *MockRequeuableObject_GetObjectKind_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRequeuableObject_GetObjectKind_Call) RunAndReturn(run func() schema.ObjectKind) *MockRequeuableObject_GetObjectKind_Call {
	_c.Call.Return(run)
	return _c
}

// GetStatus provides a mock function with given fields:
func (_m *MockRequeuableObject) GetStatus() RequeueableStatus {
	ret := _m.Called()

	var r0 RequeueableStatus
	if rf, ok := ret.Get(0).(func() RequeueableStatus); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(RequeueableStatus)
		}
	}

	return r0
}

// MockRequeuableObject_GetStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetStatus'
type MockRequeuableObject_GetStatus_Call struct {
	*mock.Call
}

// GetStatus is a helper method to define mock.On call
func (_e *MockRequeuableObject_Expecter) GetStatus() *MockRequeuableObject_GetStatus_Call {
	return &MockRequeuableObject_GetStatus_Call{Call: _e.mock.On("GetStatus")}
}

func (_c *MockRequeuableObject_GetStatus_Call) Run(run func()) *MockRequeuableObject_GetStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRequeuableObject_GetStatus_Call) Return(_a0 RequeueableStatus) *MockRequeuableObject_GetStatus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRequeuableObject_GetStatus_Call) RunAndReturn(run func() RequeueableStatus) *MockRequeuableObject_GetStatus_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewMockRequeuableObject interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockRequeuableObject creates a new instance of MockRequeuableObject. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockRequeuableObject(t mockConstructorTestingTNewMockRequeuableObject) *MockRequeuableObject {
	mock := &MockRequeuableObject{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
