// Code generated by mockery v2.42.1. DO NOT EDIT.

package requeue

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// mockRequeuableError is an autogenerated mock type for the requeuableError type
type mockRequeuableError struct {
	mock.Mock
}

type mockRequeuableError_Expecter struct {
	mock *mock.Mock
}

func (_m *mockRequeuableError) EXPECT() *mockRequeuableError_Expecter {
	return &mockRequeuableError_Expecter{mock: &_m.Mock}
}

// Error provides a mock function with given fields:
func (_m *mockRequeuableError) Error() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Error")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// mockRequeuableError_Error_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Error'
type mockRequeuableError_Error_Call struct {
	*mock.Call
}

// Error is a helper method to define mock.On call
func (_e *mockRequeuableError_Expecter) Error() *mockRequeuableError_Error_Call {
	return &mockRequeuableError_Error_Call{Call: _e.mock.On("Error")}
}

func (_c *mockRequeuableError_Error_Call) Run(run func()) *mockRequeuableError_Error_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockRequeuableError_Error_Call) Return(_a0 string) *mockRequeuableError_Error_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRequeuableError_Error_Call) RunAndReturn(run func() string) *mockRequeuableError_Error_Call {
	_c.Call.Return(run)
	return _c
}

// GetRequeueTime provides a mock function with given fields: requeueTimeNanos
func (_m *mockRequeuableError) GetRequeueTime(requeueTimeNanos time.Duration) time.Duration {
	ret := _m.Called(requeueTimeNanos)

	if len(ret) == 0 {
		panic("no return value specified for GetRequeueTime")
	}

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func(time.Duration) time.Duration); ok {
		r0 = rf(requeueTimeNanos)
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// mockRequeuableError_GetRequeueTime_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetRequeueTime'
type mockRequeuableError_GetRequeueTime_Call struct {
	*mock.Call
}

// GetRequeueTime is a helper method to define mock.On call
//   - requeueTimeNanos time.Duration
func (_e *mockRequeuableError_Expecter) GetRequeueTime(requeueTimeNanos interface{}) *mockRequeuableError_GetRequeueTime_Call {
	return &mockRequeuableError_GetRequeueTime_Call{Call: _e.mock.On("GetRequeueTime", requeueTimeNanos)}
}

func (_c *mockRequeuableError_GetRequeueTime_Call) Run(run func(requeueTimeNanos time.Duration)) *mockRequeuableError_GetRequeueTime_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(time.Duration))
	})
	return _c
}

func (_c *mockRequeuableError_GetRequeueTime_Call) Return(_a0 time.Duration) *mockRequeuableError_GetRequeueTime_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockRequeuableError_GetRequeueTime_Call) RunAndReturn(run func(time.Duration) time.Duration) *mockRequeuableError_GetRequeueTime_Call {
	_c.Call.Return(run)
	return _c
}

// newMockRequeuableError creates a new instance of mockRequeuableError. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockRequeuableError(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockRequeuableError {
	mock := &mockRequeuableError{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
