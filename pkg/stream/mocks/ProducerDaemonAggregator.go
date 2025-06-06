// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	stream "github.com/justtrackio/gosoline/pkg/stream"
	mock "github.com/stretchr/testify/mock"
)

// ProducerDaemonAggregator is an autogenerated mock type for the ProducerDaemonAggregator type
type ProducerDaemonAggregator struct {
	mock.Mock
}

type ProducerDaemonAggregator_Expecter struct {
	mock *mock.Mock
}

func (_m *ProducerDaemonAggregator) EXPECT() *ProducerDaemonAggregator_Expecter {
	return &ProducerDaemonAggregator_Expecter{mock: &_m.Mock}
}

// Flush provides a mock function with no fields
func (_m *ProducerDaemonAggregator) Flush() ([]stream.AggregateFlush, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Flush")
	}

	var r0 []stream.AggregateFlush
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]stream.AggregateFlush, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []stream.AggregateFlush); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]stream.AggregateFlush)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProducerDaemonAggregator_Flush_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Flush'
type ProducerDaemonAggregator_Flush_Call struct {
	*mock.Call
}

// Flush is a helper method to define mock.On call
func (_e *ProducerDaemonAggregator_Expecter) Flush() *ProducerDaemonAggregator_Flush_Call {
	return &ProducerDaemonAggregator_Flush_Call{Call: _e.mock.On("Flush")}
}

func (_c *ProducerDaemonAggregator_Flush_Call) Run(run func()) *ProducerDaemonAggregator_Flush_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *ProducerDaemonAggregator_Flush_Call) Return(_a0 []stream.AggregateFlush, _a1 error) *ProducerDaemonAggregator_Flush_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ProducerDaemonAggregator_Flush_Call) RunAndReturn(run func() ([]stream.AggregateFlush, error)) *ProducerDaemonAggregator_Flush_Call {
	_c.Call.Return(run)
	return _c
}

// Write provides a mock function with given fields: ctx, msg
func (_m *ProducerDaemonAggregator) Write(ctx context.Context, msg *stream.Message) ([]stream.AggregateFlush, error) {
	ret := _m.Called(ctx, msg)

	if len(ret) == 0 {
		panic("no return value specified for Write")
	}

	var r0 []stream.AggregateFlush
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *stream.Message) ([]stream.AggregateFlush, error)); ok {
		return rf(ctx, msg)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *stream.Message) []stream.AggregateFlush); ok {
		r0 = rf(ctx, msg)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]stream.AggregateFlush)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *stream.Message) error); ok {
		r1 = rf(ctx, msg)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ProducerDaemonAggregator_Write_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Write'
type ProducerDaemonAggregator_Write_Call struct {
	*mock.Call
}

// Write is a helper method to define mock.On call
//   - ctx context.Context
//   - msg *stream.Message
func (_e *ProducerDaemonAggregator_Expecter) Write(ctx interface{}, msg interface{}) *ProducerDaemonAggregator_Write_Call {
	return &ProducerDaemonAggregator_Write_Call{Call: _e.mock.On("Write", ctx, msg)}
}

func (_c *ProducerDaemonAggregator_Write_Call) Run(run func(ctx context.Context, msg *stream.Message)) *ProducerDaemonAggregator_Write_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*stream.Message))
	})
	return _c
}

func (_c *ProducerDaemonAggregator_Write_Call) Return(_a0 []stream.AggregateFlush, _a1 error) *ProducerDaemonAggregator_Write_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *ProducerDaemonAggregator_Write_Call) RunAndReturn(run func(context.Context, *stream.Message) ([]stream.AggregateFlush, error)) *ProducerDaemonAggregator_Write_Call {
	_c.Call.Return(run)
	return _c
}

// NewProducerDaemonAggregator creates a new instance of ProducerDaemonAggregator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProducerDaemonAggregator(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProducerDaemonAggregator {
	mock := &ProducerDaemonAggregator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
