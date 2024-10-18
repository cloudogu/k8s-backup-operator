// Code generated by mockery v2.42.1. DO NOT EDIT.

package ecosystem

import mock "github.com/stretchr/testify/mock"

// MockRestoresGetter is an autogenerated mock type for the RestoresGetter type
type MockRestoresGetter struct {
	mock.Mock
}

type MockRestoresGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockRestoresGetter) EXPECT() *MockRestoresGetter_Expecter {
	return &MockRestoresGetter_Expecter{mock: &_m.Mock}
}

// Restores provides a mock function with given fields: namespace
func (_m *MockRestoresGetter) Restores(namespace string) RestoreInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for Restores")
	}

	var r0 RestoreInterface
	if rf, ok := ret.Get(0).(func(string) RestoreInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(RestoreInterface)
		}
	}

	return r0
}

// MockRestoresGetter_Restores_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Restores'
type MockRestoresGetter_Restores_Call struct {
	*mock.Call
}

// Restores is a helper method to define mock.On call
//   - namespace string
func (_e *MockRestoresGetter_Expecter) Restores(namespace interface{}) *MockRestoresGetter_Restores_Call {
	return &MockRestoresGetter_Restores_Call{Call: _e.mock.On("Restores", namespace)}
}

func (_c *MockRestoresGetter_Restores_Call) Run(run func(namespace string)) *MockRestoresGetter_Restores_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockRestoresGetter_Restores_Call) Return(_a0 RestoreInterface) *MockRestoresGetter_Restores_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockRestoresGetter_Restores_Call) RunAndReturn(run func(string) RestoreInterface) *MockRestoresGetter_Restores_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockRestoresGetter creates a new instance of MockRestoresGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockRestoresGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockRestoresGetter {
	mock := &MockRestoresGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
