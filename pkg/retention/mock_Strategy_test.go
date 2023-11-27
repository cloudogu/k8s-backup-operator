// Code generated by mockery v2.20.0. DO NOT EDIT.

package retention

import (
	v1 "github.com/cloudogu/k8s-backup-operator/pkg/api/v1"
	mock "github.com/stretchr/testify/mock"
)

// MockStrategy is an autogenerated mock type for the Strategy type
type MockStrategy struct {
	mock.Mock
}

type MockStrategy_Expecter struct {
	mock *mock.Mock
}

func (_m *MockStrategy) EXPECT() *MockStrategy_Expecter {
	return &MockStrategy_Expecter{mock: &_m.Mock}
}

// FilterForRemoval provides a mock function with given fields: allBackups
func (_m *MockStrategy) FilterForRemoval(allBackups []v1.Backup) (RemovedBackups, RetainedBackups) {
	ret := _m.Called(allBackups)

	var r0 RemovedBackups
	var r1 RetainedBackups
	if rf, ok := ret.Get(0).(func([]v1.Backup) (RemovedBackups, RetainedBackups)); ok {
		return rf(allBackups)
	}
	if rf, ok := ret.Get(0).(func([]v1.Backup) RemovedBackups); ok {
		r0 = rf(allBackups)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(RemovedBackups)
		}
	}

	if rf, ok := ret.Get(1).(func([]v1.Backup) RetainedBackups); ok {
		r1 = rf(allBackups)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(RetainedBackups)
		}
	}

	return r0, r1
}

// MockStrategy_FilterForRemoval_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'FilterForRemoval'
type MockStrategy_FilterForRemoval_Call struct {
	*mock.Call
}

// FilterForRemoval is a helper method to define mock.On call
//   - allBackups []v1.Backup
func (_e *MockStrategy_Expecter) FilterForRemoval(allBackups interface{}) *MockStrategy_FilterForRemoval_Call {
	return &MockStrategy_FilterForRemoval_Call{Call: _e.mock.On("FilterForRemoval", allBackups)}
}

func (_c *MockStrategy_FilterForRemoval_Call) Run(run func(allBackups []v1.Backup)) *MockStrategy_FilterForRemoval_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]v1.Backup))
	})
	return _c
}

func (_c *MockStrategy_FilterForRemoval_Call) Return(_a0 RemovedBackups, _a1 RetainedBackups) *MockStrategy_FilterForRemoval_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStrategy_FilterForRemoval_Call) RunAndReturn(run func([]v1.Backup) (RemovedBackups, RetainedBackups)) *MockStrategy_FilterForRemoval_Call {
	_c.Call.Return(run)
	return _c
}

// GetName provides a mock function with given fields:
func (_m *MockStrategy) GetName() StrategyId {
	ret := _m.Called()

	var r0 StrategyId
	if rf, ok := ret.Get(0).(func() StrategyId); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(StrategyId)
	}

	return r0
}

// MockStrategy_GetName_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetName'
type MockStrategy_GetName_Call struct {
	*mock.Call
}

// GetName is a helper method to define mock.On call
func (_e *MockStrategy_Expecter) GetName() *MockStrategy_GetName_Call {
	return &MockStrategy_GetName_Call{Call: _e.mock.On("GetName")}
}

func (_c *MockStrategy_GetName_Call) Run(run func()) *MockStrategy_GetName_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStrategy_GetName_Call) Return(_a0 StrategyId) *MockStrategy_GetName_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStrategy_GetName_Call) RunAndReturn(run func() StrategyId) *MockStrategy_GetName_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTNewMockStrategy interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockStrategy creates a new instance of MockStrategy. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockStrategy(t mockConstructorTestingTNewMockStrategy) *MockStrategy {
	mock := &MockStrategy{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
