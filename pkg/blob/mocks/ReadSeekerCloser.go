// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// ReadSeekerCloser is an autogenerated mock type for the ReadSeekerCloser type
type ReadSeekerCloser struct {
	mock.Mock
}

type ReadSeekerCloser_Expecter struct {
	mock *mock.Mock
}

func (_m *ReadSeekerCloser) EXPECT() *ReadSeekerCloser_Expecter {
	return &ReadSeekerCloser_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with no fields
func (_m *ReadSeekerCloser) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReadSeekerCloser_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type ReadSeekerCloser_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *ReadSeekerCloser_Expecter) Close() *ReadSeekerCloser_Close_Call {
	return &ReadSeekerCloser_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *ReadSeekerCloser_Close_Call) Run(run func()) *ReadSeekerCloser_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ReadSeekerCloser_Close_Call) Return(_a0 error) *ReadSeekerCloser_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *ReadSeekerCloser_Close_Call) RunAndReturn(run func() error) *ReadSeekerCloser_Close_Call {
	_c.Call.Return(run)
	return _c
}

// Read provides a mock function with given fields: p
func (_m *ReadSeekerCloser) Read(p []byte) (int, error) {
	ret := _m.Called(p)

	if len(ret) == 0 {
		panic("no return value specified for Read")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func([]byte) (int, error)); ok {
		return rf(p)
	}
	if rf, ok := ret.Get(0).(func([]byte) int); ok {
		r0 = rf(p)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func([]byte) error); ok {
		r1 = rf(p)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadSeekerCloser_Read_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Read'
type ReadSeekerCloser_Read_Call struct {
	*mock.Call
}

// Read is a helper method to define mock.On call
//   - p []byte
func (_e *ReadSeekerCloser_Expecter) Read(p interface{}) *ReadSeekerCloser_Read_Call {
	return &ReadSeekerCloser_Read_Call{Call: _e.mock.On("Read", p)}
}

func (_c *ReadSeekerCloser_Read_Call) Run(run func(p []byte)) *ReadSeekerCloser_Read_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].([]byte))
	})
	return _c
}

func (_c *ReadSeekerCloser_Read_Call) Return(n int, err error) *ReadSeekerCloser_Read_Call {
	_c.Call.Return(n, err)
	return _c
}

func (_c *ReadSeekerCloser_Read_Call) RunAndReturn(run func([]byte) (int, error)) *ReadSeekerCloser_Read_Call {
	_c.Call.Return(run)
	return _c
}

// Seek provides a mock function with given fields: offset, whence
func (_m *ReadSeekerCloser) Seek(offset int64, whence int) (int64, error) {
	ret := _m.Called(offset, whence)

	if len(ret) == 0 {
		panic("no return value specified for Seek")
	}

	var r0 int64
	var r1 error
	if rf, ok := ret.Get(0).(func(int64, int) (int64, error)); ok {
		return rf(offset, whence)
	}
	if rf, ok := ret.Get(0).(func(int64, int) int64); ok {
		r0 = rf(offset, whence)
	} else {
		r0 = ret.Get(0).(int64)
	}

	if rf, ok := ret.Get(1).(func(int64, int) error); ok {
		r1 = rf(offset, whence)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadSeekerCloser_Seek_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Seek'
type ReadSeekerCloser_Seek_Call struct {
	*mock.Call
}

// Seek is a helper method to define mock.On call
//   - offset int64
//   - whence int
func (_e *ReadSeekerCloser_Expecter) Seek(offset interface{}, whence interface{}) *ReadSeekerCloser_Seek_Call {
	return &ReadSeekerCloser_Seek_Call{Call: _e.mock.On("Seek", offset, whence)}
}

func (_c *ReadSeekerCloser_Seek_Call) Run(run func(offset int64, whence int)) *ReadSeekerCloser_Seek_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int64), args[1].(int))
	})
	return _c
}

func (_c *ReadSeekerCloser_Seek_Call) Return(_a0 int64, _a1 error) *ReadSeekerCloser_Seek_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ReadSeekerCloser_Seek_Call) RunAndReturn(run func(int64, int) (int64, error)) *ReadSeekerCloser_Seek_Call {
	_c.Call.Return(run)
	return _c
}

// NewReadSeekerCloser creates a new instance of ReadSeekerCloser. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReadSeekerCloser(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReadSeekerCloser {
	mock := &ReadSeekerCloser{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
