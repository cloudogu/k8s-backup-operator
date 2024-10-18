// Code generated by mockery v2.42.1. DO NOT EDIT.

package v1

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockRequeueableStatus is an autogenerated mock type for the RequeueableStatus type
type MockRequeueableStatus struct {
	mock.Mock
}

type MockRequeueableStatus_Expecter struct {
	mock *mock.Mock
}

func (_m *MockRequeueableStatus) EXPECT() *MockRequeueableStatus_Expecter {
	return &MockRequeueableStatus_Expecter{mock: &_m.Mock}
}

// GetRequeueTimeNanos provides a mock function with given fields:
func (_m *MockRequeueableStatus) GetRequeueTimeNanos() time.Duration {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetRequeueTimeNanos")
	}

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func() time.Duration); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// MockRequeueableStatus_GetRequeueTimeNanos_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetRequeueTimeNanos'
type MockRequeueableStatus_GetRequeueTimeNanos_Call struct {
	*mock.Call
}

// GetRequeueTimeNanos is a helper method to define mock.On call
func (_e *MockRequeueableStatus_Expecter) GetRequeueTimeNanos() *MockRequeueableStatus_GetRequeueTimeNanos_Call {
	return &MockRequeueableStatus_GetRequeueTimeNanos_Call{Call: _e.mock.On("GetRequeueTimeNanos")}
}

func (_c *MockRequeueableStatus_GetRequeueTimeNanos_Call) Run(run func()) *MockRequeueableStatus_GetRequeueTimeNanos_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRequeueableStatus_GetRequeueTimeNanos_Call) Return(_a0 time.Duration) *MockRequeueableStatus_GetRequeueTimeNanos_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRequeueableStatus_GetRequeueTimeNanos_Call) RunAndReturn(run func() time.Duration) *MockRequeueableStatus_GetRequeueTimeNanos_Call {
	_c.Call.Return(run)
	return _c
}

// GetStatus provides a mock function with given fields:
func (_m *MockRequeueableStatus) GetStatus() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetStatus")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// MockRequeueableStatus_GetStatus_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetStatus'
type MockRequeueableStatus_GetStatus_Call struct {
	*mock.Call
}

// GetStatus is a helper method to define mock.On call
func (_e *MockRequeueableStatus_Expecter) GetStatus() *MockRequeueableStatus_GetStatus_Call {
	return &MockRequeueableStatus_GetStatus_Call{Call: _e.mock.On("GetStatus")}
}

func (_c *MockRequeueableStatus_GetStatus_Call) Run(run func()) *MockRequeueableStatus_GetStatus_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockRequeueableStatus_GetStatus_Call) Return(_a0 string) *MockRequeueableStatus_GetStatus_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRequeueableStatus_GetStatus_Call) RunAndReturn(run func() string) *MockRequeueableStatus_GetStatus_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockRequeueableStatus creates a new instance of MockRequeueableStatus. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockRequeueableStatus(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockRequeueableStatus {
	mock := &MockRequeueableStatus{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
