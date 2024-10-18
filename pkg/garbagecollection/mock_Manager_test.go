// Code generated by mockery v2.42.1. DO NOT EDIT.

package garbagecollection

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockManager is an autogenerated mock type for the Manager type
type MockManager struct {
	mock.Mock
}

type MockManager_Expecter struct {
	mock *mock.Mock
}

func (_m *MockManager) EXPECT() *MockManager_Expecter {
	return &MockManager_Expecter{mock: &_m.Mock}
}

// CollectGarbage provides a mock function with given fields: ctx
func (_m *MockManager) CollectGarbage(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for CollectGarbage")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockManager_CollectGarbage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CollectGarbage'
type MockManager_CollectGarbage_Call struct {
	*mock.Call
}

// CollectGarbage is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockManager_Expecter) CollectGarbage(ctx interface{}) *MockManager_CollectGarbage_Call {
	return &MockManager_CollectGarbage_Call{Call: _e.mock.On("CollectGarbage", ctx)}
}

func (_c *MockManager_CollectGarbage_Call) Run(run func(ctx context.Context)) *MockManager_CollectGarbage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockManager_CollectGarbage_Call) Return(_a0 error) *MockManager_CollectGarbage_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockManager_CollectGarbage_Call) RunAndReturn(run func(context.Context) error) *MockManager_CollectGarbage_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockManager creates a new instance of MockManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockManager {
	mock := &MockManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
