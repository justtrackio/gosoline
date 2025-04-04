// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	httpserver "github.com/justtrackio/gosoline/pkg/httpserver"
	mock "github.com/stretchr/testify/mock"
)

// HandlerWithoutInput is an autogenerated mock type for the HandlerWithoutInput type
type HandlerWithoutInput struct {
	mock.Mock
}

type HandlerWithoutInput_Expecter struct {
	mock *mock.Mock
}

func (_m *HandlerWithoutInput) EXPECT() *HandlerWithoutInput_Expecter {
	return &HandlerWithoutInput_Expecter{mock: &_m.Mock}
}

// Handle provides a mock function with given fields: requestContext, request
func (_m *HandlerWithoutInput) Handle(requestContext context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	ret := _m.Called(requestContext, request)

	if len(ret) == 0 {
		panic("no return value specified for Handle")
	}

	var r0 *httpserver.Response
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *httpserver.Request) (*httpserver.Response, error)); ok {
		return rf(requestContext, request)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *httpserver.Request) *httpserver.Response); ok {
		r0 = rf(requestContext, request)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*httpserver.Response)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *httpserver.Request) error); ok {
		r1 = rf(requestContext, request)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HandlerWithoutInput_Handle_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Handle'
type HandlerWithoutInput_Handle_Call struct {
	*mock.Call
}

// Handle is a helper method to define mock.On call
//   - requestContext context.Context
//   - request *httpserver.Request
func (_e *HandlerWithoutInput_Expecter) Handle(requestContext interface{}, request interface{}) *HandlerWithoutInput_Handle_Call {
	return &HandlerWithoutInput_Handle_Call{Call: _e.mock.On("Handle", requestContext, request)}
}

func (_c *HandlerWithoutInput_Handle_Call) Run(run func(requestContext context.Context, request *httpserver.Request)) *HandlerWithoutInput_Handle_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*httpserver.Request))
	})
	return _c
}

func (_c *HandlerWithoutInput_Handle_Call) Return(response *httpserver.Response, err error) *HandlerWithoutInput_Handle_Call {
	_c.Call.Return(response, err)
	return _c
}

func (_c *HandlerWithoutInput_Handle_Call) RunAndReturn(run func(context.Context, *httpserver.Request) (*httpserver.Response, error)) *HandlerWithoutInput_Handle_Call {
	_c.Call.Return(run)
	return _c
}

// NewHandlerWithoutInput creates a new instance of HandlerWithoutInput. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewHandlerWithoutInput(t interface {
	mock.TestingT
	Cleanup(func())
}) *HandlerWithoutInput {
	mock := &HandlerWithoutInput{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
