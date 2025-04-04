// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// RunnableConsumerCallback is an autogenerated mock type for the RunnableConsumerCallback type
type RunnableConsumerCallback struct {
	mock.Mock
}

type RunnableConsumerCallback_Expecter struct {
	mock *mock.Mock
}

func (_m *RunnableConsumerCallback) EXPECT() *RunnableConsumerCallback_Expecter {
	return &RunnableConsumerCallback_Expecter{mock: &_m.Mock}
}

// Consume provides a mock function with given fields: ctx, model, attributes
func (_m *RunnableConsumerCallback) Consume(ctx context.Context, model interface{}, attributes map[string]string) (bool, error) {
	ret := _m.Called(ctx, model, attributes)

	if len(ret) == 0 {
		panic("no return value specified for Consume")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, interface{}, map[string]string) (bool, error)); ok {
		return rf(ctx, model, attributes)
	}
	if rf, ok := ret.Get(0).(func(context.Context, interface{}, map[string]string) bool); ok {
		r0 = rf(ctx, model, attributes)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, interface{}, map[string]string) error); ok {
		r1 = rf(ctx, model, attributes)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RunnableConsumerCallback_Consume_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Consume'
type RunnableConsumerCallback_Consume_Call struct {
	*mock.Call
}

// Consume is a helper method to define mock.On call
//   - ctx context.Context
//   - model interface{}
//   - attributes map[string]string
func (_e *RunnableConsumerCallback_Expecter) Consume(ctx interface{}, model interface{}, attributes interface{}) *RunnableConsumerCallback_Consume_Call {
	return &RunnableConsumerCallback_Consume_Call{Call: _e.mock.On("Consume", ctx, model, attributes)}
}

func (_c *RunnableConsumerCallback_Consume_Call) Run(run func(ctx context.Context, model interface{}, attributes map[string]string)) *RunnableConsumerCallback_Consume_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(interface{}), args[2].(map[string]string))
	})
	return _c
}

func (_c *RunnableConsumerCallback_Consume_Call) Return(_a0 bool, _a1 error) *RunnableConsumerCallback_Consume_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *RunnableConsumerCallback_Consume_Call) RunAndReturn(run func(context.Context, interface{}, map[string]string) (bool, error)) *RunnableConsumerCallback_Consume_Call {
	_c.Call.Return(run)
	return _c
}

// GetModel provides a mock function with given fields: attributes
func (_m *RunnableConsumerCallback) GetModel(attributes map[string]string) interface{} {
	ret := _m.Called(attributes)

	if len(ret) == 0 {
		panic("no return value specified for GetModel")
	}

	var r0 interface{}
	if rf, ok := ret.Get(0).(func(map[string]string) interface{}); ok {
		r0 = rf(attributes)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	return r0
}

// RunnableConsumerCallback_GetModel_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetModel'
type RunnableConsumerCallback_GetModel_Call struct {
	*mock.Call
}

// GetModel is a helper method to define mock.On call
//   - attributes map[string]string
func (_e *RunnableConsumerCallback_Expecter) GetModel(attributes interface{}) *RunnableConsumerCallback_GetModel_Call {
	return &RunnableConsumerCallback_GetModel_Call{Call: _e.mock.On("GetModel", attributes)}
}

func (_c *RunnableConsumerCallback_GetModel_Call) Run(run func(attributes map[string]string)) *RunnableConsumerCallback_GetModel_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(map[string]string))
	})
	return _c
}

func (_c *RunnableConsumerCallback_GetModel_Call) Return(_a0 interface{}) *RunnableConsumerCallback_GetModel_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *RunnableConsumerCallback_GetModel_Call) RunAndReturn(run func(map[string]string) interface{}) *RunnableConsumerCallback_GetModel_Call {
	_c.Call.Return(run)
	return _c
}

// Run provides a mock function with given fields: ctx
func (_m *RunnableConsumerCallback) Run(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Run")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RunnableConsumerCallback_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type RunnableConsumerCallback_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
//   - ctx context.Context
func (_e *RunnableConsumerCallback_Expecter) Run(ctx interface{}) *RunnableConsumerCallback_Run_Call {
	return &RunnableConsumerCallback_Run_Call{Call: _e.mock.On("Run", ctx)}
}

func (_c *RunnableConsumerCallback_Run_Call) Run(run func(ctx context.Context)) *RunnableConsumerCallback_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *RunnableConsumerCallback_Run_Call) Return(_a0 error) *RunnableConsumerCallback_Run_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *RunnableConsumerCallback_Run_Call) RunAndReturn(run func(context.Context) error) *RunnableConsumerCallback_Run_Call {
	_c.Call.Return(run)
	return _c
}

// NewRunnableConsumerCallback creates a new instance of RunnableConsumerCallback. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRunnableConsumerCallback(t interface {
	mock.TestingT
	Cleanup(func())
}) *RunnableConsumerCallback {
	mock := &RunnableConsumerCallback{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
