// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	metric "github.com/justtrackio/gosoline/pkg/metric"
	mock "github.com/stretchr/testify/mock"
)

// Handler is an autogenerated mock type for the Handler type
type Handler struct {
	mock.Mock
}

type Handler_Expecter struct {
	mock *mock.Mock
}

func (_m *Handler) EXPECT() *Handler_Expecter {
	return &Handler_Expecter{mock: &_m.Mock}
}

// GetMetrics provides a mock function with given fields: ctx
func (_m *Handler) GetMetrics(ctx context.Context) (metric.Data, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetMetrics")
	}

	var r0 metric.Data
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (metric.Data, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) metric.Data); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(metric.Data)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Handler_GetMetrics_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetMetrics'
type Handler_GetMetrics_Call struct {
	*mock.Call
}

// GetMetrics is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Handler_Expecter) GetMetrics(ctx interface{}) *Handler_GetMetrics_Call {
	return &Handler_GetMetrics_Call{Call: _e.mock.On("GetMetrics", ctx)}
}

func (_c *Handler_GetMetrics_Call) Run(run func(ctx context.Context)) *Handler_GetMetrics_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Handler_GetMetrics_Call) Return(_a0 metric.Data, _a1 error) *Handler_GetMetrics_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Handler_GetMetrics_Call) RunAndReturn(run func(context.Context) (metric.Data, error)) *Handler_GetMetrics_Call {
	_c.Call.Return(run)
	return _c
}

// NewHandler creates a new instance of Handler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *Handler {
	mock := &Handler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
