// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// UpdaterService is an autogenerated mock type for the UpdaterService type
type UpdaterService struct {
	mock.Mock
}

type UpdaterService_Expecter struct {
	mock *mock.Mock
}

func (_m *UpdaterService) EXPECT() *UpdaterService_Expecter {
	return &UpdaterService_Expecter{mock: &_m.Mock}
}

// EnsureHistoricalExchangeRates provides a mock function with given fields: ctx
func (_m *UpdaterService) EnsureHistoricalExchangeRates(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for EnsureHistoricalExchangeRates")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdaterService_EnsureHistoricalExchangeRates_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EnsureHistoricalExchangeRates'
type UpdaterService_EnsureHistoricalExchangeRates_Call struct {
	*mock.Call
}

// EnsureHistoricalExchangeRates is a helper method to define mock.On call
//   - ctx context.Context
func (_e *UpdaterService_Expecter) EnsureHistoricalExchangeRates(ctx interface{}) *UpdaterService_EnsureHistoricalExchangeRates_Call {
	return &UpdaterService_EnsureHistoricalExchangeRates_Call{Call: _e.mock.On("EnsureHistoricalExchangeRates", ctx)}
}

func (_c *UpdaterService_EnsureHistoricalExchangeRates_Call) Run(run func(ctx context.Context)) *UpdaterService_EnsureHistoricalExchangeRates_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *UpdaterService_EnsureHistoricalExchangeRates_Call) Return(_a0 error) *UpdaterService_EnsureHistoricalExchangeRates_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *UpdaterService_EnsureHistoricalExchangeRates_Call) RunAndReturn(run func(context.Context) error) *UpdaterService_EnsureHistoricalExchangeRates_Call {
	_c.Call.Return(run)
	return _c
}

// EnsureRecentExchangeRates provides a mock function with given fields: ctx
func (_m *UpdaterService) EnsureRecentExchangeRates(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for EnsureRecentExchangeRates")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdaterService_EnsureRecentExchangeRates_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'EnsureRecentExchangeRates'
type UpdaterService_EnsureRecentExchangeRates_Call struct {
	*mock.Call
}

// EnsureRecentExchangeRates is a helper method to define mock.On call
//   - ctx context.Context
func (_e *UpdaterService_Expecter) EnsureRecentExchangeRates(ctx interface{}) *UpdaterService_EnsureRecentExchangeRates_Call {
	return &UpdaterService_EnsureRecentExchangeRates_Call{Call: _e.mock.On("EnsureRecentExchangeRates", ctx)}
}

func (_c *UpdaterService_EnsureRecentExchangeRates_Call) Run(run func(ctx context.Context)) *UpdaterService_EnsureRecentExchangeRates_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *UpdaterService_EnsureRecentExchangeRates_Call) Return(_a0 error) *UpdaterService_EnsureRecentExchangeRates_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *UpdaterService_EnsureRecentExchangeRates_Call) RunAndReturn(run func(context.Context) error) *UpdaterService_EnsureRecentExchangeRates_Call {
	_c.Call.Return(run)
	return _c
}

// NewUpdaterService creates a new instance of UpdaterService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUpdaterService(t interface {
	mock.TestingT
	Cleanup(func())
}) *UpdaterService {
	mock := &UpdaterService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
