// Code generated by mockery v2.53.3. DO NOT EDIT.

package ecosystem

import mock "github.com/stretchr/testify/mock"

// MockBackupSchedulesGetter is an autogenerated mock type for the BackupSchedulesGetter type
type MockBackupSchedulesGetter struct {
	mock.Mock
}

type MockBackupSchedulesGetter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockBackupSchedulesGetter) EXPECT() *MockBackupSchedulesGetter_Expecter {
	return &MockBackupSchedulesGetter_Expecter{mock: &_m.Mock}
}

// BackupSchedules provides a mock function with given fields: namespace
func (_m *MockBackupSchedulesGetter) BackupSchedules(namespace string) BackupScheduleInterface {
	ret := _m.Called(namespace)

	if len(ret) == 0 {
		panic("no return value specified for BackupSchedules")
	}

	var r0 BackupScheduleInterface
	if rf, ok := ret.Get(0).(func(string) BackupScheduleInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(BackupScheduleInterface)
		}
	}

	return r0
}

// MockBackupSchedulesGetter_BackupSchedules_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'BackupSchedules'
type MockBackupSchedulesGetter_BackupSchedules_Call struct {
	*mock.Call
}

// BackupSchedules is a helper method to define mock.On call
//   - namespace string
func (_e *MockBackupSchedulesGetter_Expecter) BackupSchedules(namespace interface{}) *MockBackupSchedulesGetter_BackupSchedules_Call {
	return &MockBackupSchedulesGetter_BackupSchedules_Call{Call: _e.mock.On("BackupSchedules", namespace)}
}

func (_c *MockBackupSchedulesGetter_BackupSchedules_Call) Run(run func(namespace string)) *MockBackupSchedulesGetter_BackupSchedules_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *MockBackupSchedulesGetter_BackupSchedules_Call) Return(_a0 BackupScheduleInterface) *MockBackupSchedulesGetter_BackupSchedules_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockBackupSchedulesGetter_BackupSchedules_Call) RunAndReturn(run func(string) BackupScheduleInterface) *MockBackupSchedulesGetter_BackupSchedules_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockBackupSchedulesGetter creates a new instance of MockBackupSchedulesGetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockBackupSchedulesGetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockBackupSchedulesGetter {
	mock := &MockBackupSchedulesGetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
