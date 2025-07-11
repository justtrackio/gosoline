// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// UntypedConsumerCallback is an autogenerated mock type for the UntypedConsumerCallback type
type UntypedConsumerCallback struct {
	mock.Mock
}

type UntypedConsumerCallback_Expecter struct {
	mock *mock.Mock
}

func (_m *UntypedConsumerCallback) EXPECT() *UntypedConsumerCallback_Expecter {
	return &UntypedConsumerCallback_Expecter{mock: &_m.Mock}
}

// Consume provides a mock function with given fields: ctx, model, attributes
func (_m *UntypedConsumerCallback) Consume(ctx context.Context, model interface{}, attributes map[string]string) (bool, error) {
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

// UntypedConsumerCallback_Consume_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Consume'
type UntypedConsumerCallback_Consume_Call struct {
	*mock.Call
}

// Consume is a helper method to define mock.On call
//   - ctx context.Context
//   - model interface{}
//   - attributes map[string]string
func (_e *UntypedConsumerCallback_Expecter) Consume(ctx interface{}, model interface{}, attributes interface{}) *UntypedConsumerCallback_Consume_Call {
	return &UntypedConsumerCallback_Consume_Call{Call: _e.mock.On("Consume", ctx, model, attributes)}
}

func (_c *UntypedConsumerCallback_Consume_Call) Run(run func(ctx context.Context, model interface{}, attributes map[string]string)) *UntypedConsumerCallback_Consume_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(interface{}), args[2].(map[string]string))
	})
	return _c
}

func (_c *UntypedConsumerCallback_Consume_Call) Return(_a0 bool, _a1 error) *UntypedConsumerCallback_Consume_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *UntypedConsumerCallback_Consume_Call) RunAndReturn(run func(context.Context, interface{}, map[string]string) (bool, error)) *UntypedConsumerCallback_Consume_Call {
	_c.Call.Return(run)
	return _c
}

// GetModel provides a mock function with given fields: attributes
func (_m *UntypedConsumerCallback) GetModel(attributes map[string]string) interface{} {
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

// UntypedConsumerCallback_GetModel_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetModel'
type UntypedConsumerCallback_GetModel_Call struct {
	*mock.Call
}

// GetModel is a helper method to define mock.On call
//   - attributes map[string]string
func (_e *UntypedConsumerCallback_Expecter) GetModel(attributes interface{}) *UntypedConsumerCallback_GetModel_Call {
	return &UntypedConsumerCallback_GetModel_Call{Call: _e.mock.On("GetModel", attributes)}
}

func (_c *UntypedConsumerCallback_GetModel_Call) Run(run func(attributes map[string]string)) *UntypedConsumerCallback_GetModel_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(map[string]string))
	})
	return _c
}

func (_c *UntypedConsumerCallback_GetModel_Call) Return(_a0 interface{}) *UntypedConsumerCallback_GetModel_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *UntypedConsumerCallback_GetModel_Call) RunAndReturn(run func(map[string]string) interface{}) *UntypedConsumerCallback_GetModel_Call {
	_c.Call.Return(run)
	return _c
}

// NewUntypedConsumerCallback creates a new instance of UntypedConsumerCallback. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUntypedConsumerCallback(t interface {
	mock.TestingT
	Cleanup(func())
}) *UntypedConsumerCallback {
	mock := &UntypedConsumerCallback{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
