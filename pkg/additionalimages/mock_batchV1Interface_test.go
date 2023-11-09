// Code generated by mockery v2.20.0. DO NOT EDIT.

package additionalimages

import (
	mock "github.com/stretchr/testify/mock"
	rest "k8s.io/client-go/rest"

	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
)

// mockBatchV1Interface is an autogenerated mock type for the batchV1Interface type
type mockBatchV1Interface struct {
	mock.Mock
}

type mockBatchV1Interface_Expecter struct {
	mock *mock.Mock
}

func (_m *mockBatchV1Interface) EXPECT() *mockBatchV1Interface_Expecter {
	return &mockBatchV1Interface_Expecter{mock: &_m.Mock}
}

// CronJobs provides a mock function with given fields: namespace
func (_m *mockBatchV1Interface) CronJobs(namespace string) v1.CronJobInterface {
	ret := _m.Called(namespace)

	var r0 v1.CronJobInterface
	if rf, ok := ret.Get(0).(func(string) v1.CronJobInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.CronJobInterface)
		}
	}

	return r0
}

// mockBatchV1Interface_CronJobs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'CronJobs'
type mockBatchV1Interface_CronJobs_Call struct {
	*mock.Call
}

// CronJobs is a helper method to define mock.On call
//   - namespace string
func (_e *mockBatchV1Interface_Expecter) CronJobs(namespace interface{}) *mockBatchV1Interface_CronJobs_Call {
	return &mockBatchV1Interface_CronJobs_Call{Call: _e.mock.On("CronJobs", namespace)}
}

func (_c *mockBatchV1Interface_CronJobs_Call) Run(run func(namespace string)) *mockBatchV1Interface_CronJobs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockBatchV1Interface_CronJobs_Call) Return(_a0 v1.CronJobInterface) *mockBatchV1Interface_CronJobs_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockBatchV1Interface_CronJobs_Call) RunAndReturn(run func(string) v1.CronJobInterface) *mockBatchV1Interface_CronJobs_Call {
	_c.Call.Return(run)
	return _c
}

// Jobs provides a mock function with given fields: namespace
func (_m *mockBatchV1Interface) Jobs(namespace string) v1.JobInterface {
	ret := _m.Called(namespace)

	var r0 v1.JobInterface
	if rf, ok := ret.Get(0).(func(string) v1.JobInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1.JobInterface)
		}
	}

	return r0
}

// mockBatchV1Interface_Jobs_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Jobs'
type mockBatchV1Interface_Jobs_Call struct {
	*mock.Call
}

// Jobs is a helper method to define mock.On call
//   - namespace string
func (_e *mockBatchV1Interface_Expecter) Jobs(namespace interface{}) *mockBatchV1Interface_Jobs_Call {
	return &mockBatchV1Interface_Jobs_Call{Call: _e.mock.On("Jobs", namespace)}
}

func (_c *mockBatchV1Interface_Jobs_Call) Run(run func(namespace string)) *mockBatchV1Interface_Jobs_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *mockBatchV1Interface_Jobs_Call) Return(_a0 v1.JobInterface) *mockBatchV1Interface_Jobs_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockBatchV1Interface_Jobs_Call) RunAndReturn(run func(string) v1.JobInterface) *mockBatchV1Interface_Jobs_Call {
	_c.Call.Return(run)
	return _c
}

// RESTClient provides a mock function with given fields:
func (_m *mockBatchV1Interface) RESTClient() rest.Interface {
	ret := _m.Called()

	var r0 rest.Interface
	if rf, ok := ret.Get(0).(func() rest.Interface); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(rest.Interface)
		}
	}

	return r0
}

// mockBatchV1Interface_RESTClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RESTClient'
type mockBatchV1Interface_RESTClient_Call struct {
	*mock.Call
}

// RESTClient is a helper method to define mock.On call
func (_e *mockBatchV1Interface_Expecter) RESTClient() *mockBatchV1Interface_RESTClient_Call {
	return &mockBatchV1Interface_RESTClient_Call{Call: _e.mock.On("RESTClient")}
}

func (_c *mockBatchV1Interface_RESTClient_Call) Run(run func()) *mockBatchV1Interface_RESTClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *mockBatchV1Interface_RESTClient_Call) Return(_a0 rest.Interface) *mockBatchV1Interface_RESTClient_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *mockBatchV1Interface_RESTClient_Call) RunAndReturn(run func() rest.Interface) *mockBatchV1Interface_RESTClient_Call {
	_c.Call.Return(run)
	return _c
}

type mockConstructorTestingTnewMockBatchV1Interface interface {
	mock.TestingT
	Cleanup(func())
}

// newMockBatchV1Interface creates a new instance of mockBatchV1Interface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockBatchV1Interface(t mockConstructorTestingTnewMockBatchV1Interface) *mockBatchV1Interface {
	mock := &mockBatchV1Interface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
