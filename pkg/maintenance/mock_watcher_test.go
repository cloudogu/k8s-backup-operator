// Code generated by mockery v2.42.1. DO NOT EDIT.

package maintenance

import (
	mock "github.com/stretchr/testify/mock"
	watch "k8s.io/apimachinery/pkg/watch"
)

// mockWatcher is an autogenerated mock type for the watcher type
type mockWatcher struct {
	mock.Mock
}

type mockWatcher_Expecter struct {
	mock *mock.Mock
}

func (_m *mockWatcher) EXPECT() *mockWatcher_Expecter {
	return &mockWatcher_Expecter{mock: &_m.Mock}
}

// ResultChan provides a mock function with given fields:
func (_m *mockWatcher) ResultChan() <-chan watch.Event {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ResultChan")
	}

	var r0 <-chan watch.Event
	if rf, ok := ret.Get(0).(func() <-chan watch.Event); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan watch.Event)
		}
	}

	return r0
}

// mockWatcher_ResultChan_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ResultChan'
type mockWatcher_ResultChan_Call struct {
	*mock.Call
}

// ResultChan is a helper method to define mock.On call
func (_e *mockWatcher_Expecter) ResultChan() *mockWatcher_ResultChan_Call {
	return &mockWatcher_ResultChan_Call{Call: _e.mock.On("ResultChan")}
}

func (_c *mockWatcher_ResultChan_Call) Run(run func()) *mockWatcher_ResultChan_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockWatcher_ResultChan_Call) Return(_a0 <-chan watch.Event) *mockWatcher_ResultChan_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockWatcher_ResultChan_Call) RunAndReturn(run func() <-chan watch.Event) *mockWatcher_ResultChan_Call {
	_c.Call.Return(run)
	return _c
}

// Stop provides a mock function with given fields:
func (_m *mockWatcher) Stop() {
	_m.Called()
}

// mockWatcher_Stop_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stop'
type mockWatcher_Stop_Call struct {
	*mock.Call
}

// Stop is a helper method to define mock.On call
func (_e *mockWatcher_Expecter) Stop() *mockWatcher_Stop_Call {
	return &mockWatcher_Stop_Call{Call: _e.mock.On("Stop")}
}

func (_c *mockWatcher_Stop_Call) Run(run func()) *mockWatcher_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockWatcher_Stop_Call) Return() *mockWatcher_Stop_Call {
	_c.Call.Return()
	return _c
}

func (_c *mockWatcher_Stop_Call) RunAndReturn(run func()) *mockWatcher_Stop_Call {
	_c.Call.Return(run)
	return _c
}

// newMockWatcher creates a new instance of mockWatcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newMockWatcher(t interface {
	mock.TestingT
	Cleanup(func())
}) *mockWatcher {
	mock := &mockWatcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
