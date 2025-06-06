// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// UrlBuilder is an autogenerated mock type for the UrlBuilder type
type UrlBuilder struct {
	mock.Mock
}

type UrlBuilder_Expecter struct {
	mock *mock.Mock
}

func (_m *UrlBuilder) EXPECT() *UrlBuilder_Expecter {
	return &UrlBuilder_Expecter{mock: &_m.Mock}
}

// GetAbsoluteUrl provides a mock function with given fields: path
func (_m *UrlBuilder) GetAbsoluteUrl(path string) (string, error) {
	ret := _m.Called(path)

	if len(ret) == 0 {
		panic("no return value specified for GetAbsoluteUrl")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (string, error)); ok {
		return rf(path)
	}
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(path)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(path)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UrlBuilder_GetAbsoluteUrl_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAbsoluteUrl'
type UrlBuilder_GetAbsoluteUrl_Call struct {
	*mock.Call
}

// GetAbsoluteUrl is a helper method to define mock.On call
//   - path string
func (_e *UrlBuilder_Expecter) GetAbsoluteUrl(path interface{}) *UrlBuilder_GetAbsoluteUrl_Call {
	return &UrlBuilder_GetAbsoluteUrl_Call{Call: _e.mock.On("GetAbsoluteUrl", path)}
}

func (_c *UrlBuilder_GetAbsoluteUrl_Call) Run(run func(path string)) *UrlBuilder_GetAbsoluteUrl_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *UrlBuilder_GetAbsoluteUrl_Call) Return(_a0 string, _a1 error) *UrlBuilder_GetAbsoluteUrl_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *UrlBuilder_GetAbsoluteUrl_Call) RunAndReturn(run func(string) (string, error)) *UrlBuilder_GetAbsoluteUrl_Call {
	_c.Call.Return(run)
	return _c
}

// NewUrlBuilder creates a new instance of UrlBuilder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUrlBuilder(t interface {
	mock.TestingT
	Cleanup(func())
}) *UrlBuilder {
	mock := &UrlBuilder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
