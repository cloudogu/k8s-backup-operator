// Code generated by mockery v2.53.3. DO NOT EDIT.

package scheduledbackup

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

// ScheduleBackup provides a mock function with given fields: ctx
func (_m *MockManager) ScheduleBackup(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for ScheduleBackup")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockManager_ScheduleBackup_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ScheduleBackup'
type MockManager_ScheduleBackup_Call struct {
	*mock.Call
}

// ScheduleBackup is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockManager_Expecter) ScheduleBackup(ctx interface{}) *MockManager_ScheduleBackup_Call {
	return &MockManager_ScheduleBackup_Call{Call: _e.mock.On("ScheduleBackup", ctx)}
}

func (_c *MockManager_ScheduleBackup_Call) Run(run func(ctx context.Context)) *MockManager_ScheduleBackup_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockManager_ScheduleBackup_Call) Return(_a0 error) *MockManager_ScheduleBackup_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockManager_ScheduleBackup_Call) RunAndReturn(run func(context.Context) error) *MockManager_ScheduleBackup_Call {
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
