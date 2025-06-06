// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// LeaderElection is an autogenerated mock type for the LeaderElection type
type LeaderElection struct {
	mock.Mock
}

type LeaderElection_Expecter struct {
	mock *mock.Mock
}

func (_m *LeaderElection) EXPECT() *LeaderElection_Expecter {
	return &LeaderElection_Expecter{mock: &_m.Mock}
}

// IsLeader provides a mock function with given fields: ctx, memberId
func (_m *LeaderElection) IsLeader(ctx context.Context, memberId string) (bool, error) {
	ret := _m.Called(ctx, memberId)

	if len(ret) == 0 {
		panic("no return value specified for IsLeader")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (bool, error)); ok {
		return rf(ctx, memberId)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, memberId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, memberId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LeaderElection_IsLeader_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'IsLeader'
type LeaderElection_IsLeader_Call struct {
	*mock.Call
}

// IsLeader is a helper method to define mock.On call
//   - ctx context.Context
//   - memberId string
func (_e *LeaderElection_Expecter) IsLeader(ctx interface{}, memberId interface{}) *LeaderElection_IsLeader_Call {
	return &LeaderElection_IsLeader_Call{Call: _e.mock.On("IsLeader", ctx, memberId)}
}

func (_c *LeaderElection_IsLeader_Call) Run(run func(ctx context.Context, memberId string)) *LeaderElection_IsLeader_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *LeaderElection_IsLeader_Call) Return(_a0 bool, _a1 error) *LeaderElection_IsLeader_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *LeaderElection_IsLeader_Call) RunAndReturn(run func(context.Context, string) (bool, error)) *LeaderElection_IsLeader_Call {
	_c.Call.Return(run)
	return _c
}

// Resign provides a mock function with given fields: ctx, memberId
func (_m *LeaderElection) Resign(ctx context.Context, memberId string) error {
	ret := _m.Called(ctx, memberId)

	if len(ret) == 0 {
		panic("no return value specified for Resign")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, memberId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LeaderElection_Resign_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Resign'
type LeaderElection_Resign_Call struct {
	*mock.Call
}

// Resign is a helper method to define mock.On call
//   - ctx context.Context
//   - memberId string
func (_e *LeaderElection_Expecter) Resign(ctx interface{}, memberId interface{}) *LeaderElection_Resign_Call {
	return &LeaderElection_Resign_Call{Call: _e.mock.On("Resign", ctx, memberId)}
}

func (_c *LeaderElection_Resign_Call) Run(run func(ctx context.Context, memberId string)) *LeaderElection_Resign_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *LeaderElection_Resign_Call) Return(_a0 error) *LeaderElection_Resign_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *LeaderElection_Resign_Call) RunAndReturn(run func(context.Context, string) error) *LeaderElection_Resign_Call {
	_c.Call.Return(run)
	return _c
}

// NewLeaderElection creates a new instance of LeaderElection. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewLeaderElection(t interface {
	mock.TestingT
	Cleanup(func())
}) *LeaderElection {
	mock := &LeaderElection{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
