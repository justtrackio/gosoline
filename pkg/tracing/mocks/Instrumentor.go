// Code generated by mockery v2.46.0. DO NOT EDIT.

package mocks

import (
	http "net/http"

	grpc "google.golang.org/grpc"

	mock "github.com/stretchr/testify/mock"

	stats "google.golang.org/grpc/stats"
)

// Instrumentor is an autogenerated mock type for the Instrumentor type
type Instrumentor struct {
	mock.Mock
}

type Instrumentor_Expecter struct {
	mock *mock.Mock
}

func (_m *Instrumentor) EXPECT() *Instrumentor_Expecter {
	return &Instrumentor_Expecter{mock: &_m.Mock}
}

// GrpcServerHandler provides a mock function with given fields:
func (_m *Instrumentor) GrpcServerHandler() stats.Handler {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GrpcServerHandler")
	}

	var r0 stats.Handler
	if rf, ok := ret.Get(0).(func() stats.Handler); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(stats.Handler)
		}
	}

	return r0
}

// Instrumentor_GrpcServerHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GrpcServerHandler'
type Instrumentor_GrpcServerHandler_Call struct {
	*mock.Call
}

// GrpcServerHandler is a helper method to define mock.On call
func (_e *Instrumentor_Expecter) GrpcServerHandler() *Instrumentor_GrpcServerHandler_Call {
	return &Instrumentor_GrpcServerHandler_Call{Call: _e.mock.On("GrpcServerHandler")}
}

func (_c *Instrumentor_GrpcServerHandler_Call) Run(run func()) *Instrumentor_GrpcServerHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Instrumentor_GrpcServerHandler_Call) Return(_a0 stats.Handler) *Instrumentor_GrpcServerHandler_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Instrumentor_GrpcServerHandler_Call) RunAndReturn(run func() stats.Handler) *Instrumentor_GrpcServerHandler_Call {
	_c.Call.Return(run)
	return _c
}

// GrpcUnaryServerInterceptor provides a mock function with given fields:
func (_m *Instrumentor) GrpcUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GrpcUnaryServerInterceptor")
	}

	var r0 grpc.UnaryServerInterceptor
	if rf, ok := ret.Get(0).(func() grpc.UnaryServerInterceptor); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(grpc.UnaryServerInterceptor)
		}
	}

	return r0
}

// Instrumentor_GrpcUnaryServerInterceptor_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GrpcUnaryServerInterceptor'
type Instrumentor_GrpcUnaryServerInterceptor_Call struct {
	*mock.Call
}

// GrpcUnaryServerInterceptor is a helper method to define mock.On call
func (_e *Instrumentor_Expecter) GrpcUnaryServerInterceptor() *Instrumentor_GrpcUnaryServerInterceptor_Call {
	return &Instrumentor_GrpcUnaryServerInterceptor_Call{Call: _e.mock.On("GrpcUnaryServerInterceptor")}
}

func (_c *Instrumentor_GrpcUnaryServerInterceptor_Call) Run(run func()) *Instrumentor_GrpcUnaryServerInterceptor_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Instrumentor_GrpcUnaryServerInterceptor_Call) Return(_a0 grpc.UnaryServerInterceptor) *Instrumentor_GrpcUnaryServerInterceptor_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Instrumentor_GrpcUnaryServerInterceptor_Call) RunAndReturn(run func() grpc.UnaryServerInterceptor) *Instrumentor_GrpcUnaryServerInterceptor_Call {
	_c.Call.Return(run)
	return _c
}

// HttpClient provides a mock function with given fields: baseClient
func (_m *Instrumentor) HttpClient(baseClient *http.Client) *http.Client {
	ret := _m.Called(baseClient)

	if len(ret) == 0 {
		panic("no return value specified for HttpClient")
	}

	var r0 *http.Client
	if rf, ok := ret.Get(0).(func(*http.Client) *http.Client); ok {
		r0 = rf(baseClient)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*http.Client)
		}
	}

	return r0
}

// Instrumentor_HttpClient_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HttpClient'
type Instrumentor_HttpClient_Call struct {
	*mock.Call
}

// HttpClient is a helper method to define mock.On call
//   - baseClient *http.Client
func (_e *Instrumentor_Expecter) HttpClient(baseClient interface{}) *Instrumentor_HttpClient_Call {
	return &Instrumentor_HttpClient_Call{Call: _e.mock.On("HttpClient", baseClient)}
}

func (_c *Instrumentor_HttpClient_Call) Run(run func(baseClient *http.Client)) *Instrumentor_HttpClient_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(*http.Client))
	})
	return _c
}

func (_c *Instrumentor_HttpClient_Call) Return(_a0 *http.Client) *Instrumentor_HttpClient_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Instrumentor_HttpClient_Call) RunAndReturn(run func(*http.Client) *http.Client) *Instrumentor_HttpClient_Call {
	_c.Call.Return(run)
	return _c
}

// HttpHandler provides a mock function with given fields: h
func (_m *Instrumentor) HttpHandler(h http.Handler) http.Handler {
	ret := _m.Called(h)

	if len(ret) == 0 {
		panic("no return value specified for HttpHandler")
	}

	var r0 http.Handler
	if rf, ok := ret.Get(0).(func(http.Handler) http.Handler); ok {
		r0 = rf(h)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(http.Handler)
		}
	}

	return r0
}

// Instrumentor_HttpHandler_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'HttpHandler'
type Instrumentor_HttpHandler_Call struct {
	*mock.Call
}

// HttpHandler is a helper method to define mock.On call
//   - h http.Handler
func (_e *Instrumentor_Expecter) HttpHandler(h interface{}) *Instrumentor_HttpHandler_Call {
	return &Instrumentor_HttpHandler_Call{Call: _e.mock.On("HttpHandler", h)}
}

func (_c *Instrumentor_HttpHandler_Call) Run(run func(h http.Handler)) *Instrumentor_HttpHandler_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(http.Handler))
	})
	return _c
}

func (_c *Instrumentor_HttpHandler_Call) Return(_a0 http.Handler) *Instrumentor_HttpHandler_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Instrumentor_HttpHandler_Call) RunAndReturn(run func(http.Handler) http.Handler) *Instrumentor_HttpHandler_Call {
	_c.Call.Return(run)
	return _c
}

// NewInstrumentor creates a new instance of Instrumentor. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewInstrumentor(t interface {
	mock.TestingT
	Cleanup(func())
}) *Instrumentor {
	mock := &Instrumentor{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}