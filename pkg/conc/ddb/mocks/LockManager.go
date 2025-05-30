// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// LockManager is an autogenerated mock type for the LockManager type
type LockManager struct {
	mock.Mock
}

type LockManager_Expecter struct {
	mock *mock.Mock
}

func (_m *LockManager) EXPECT() *LockManager_Expecter {
	return &LockManager_Expecter{mock: &_m.Mock}
}

// ReleaseLock provides a mock function with given fields: ctx, resource, token
func (_m *LockManager) ReleaseLock(ctx context.Context, resource string, token string) error {
	ret := _m.Called(ctx, resource, token)

	if len(ret) == 0 {
		panic("no return value specified for ReleaseLock")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, resource, token)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LockManager_ReleaseLock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ReleaseLock'
type LockManager_ReleaseLock_Call struct {
	*mock.Call
}

// ReleaseLock is a helper method to define mock.On call
//   - ctx context.Context
//   - resource string
//   - token string
func (_e *LockManager_Expecter) ReleaseLock(ctx interface{}, resource interface{}, token interface{}) *LockManager_ReleaseLock_Call {
	return &LockManager_ReleaseLock_Call{Call: _e.mock.On("ReleaseLock", ctx, resource, token)}
}

func (_c *LockManager_ReleaseLock_Call) Run(run func(ctx context.Context, resource string, token string)) *LockManager_ReleaseLock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *LockManager_ReleaseLock_Call) Return(_a0 error) *LockManager_ReleaseLock_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *LockManager_ReleaseLock_Call) RunAndReturn(run func(context.Context, string, string) error) *LockManager_ReleaseLock_Call {
	_c.Call.Return(run)
	return _c
}

// RenewLock provides a mock function with given fields: ctx, lockTime, resource, token
func (_m *LockManager) RenewLock(ctx context.Context, lockTime time.Duration, resource string, token string) error {
	ret := _m.Called(ctx, lockTime, resource, token)

	if len(ret) == 0 {
		panic("no return value specified for RenewLock")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Duration, string, string) error); ok {
		r0 = rf(ctx, lockTime, resource, token)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LockManager_RenewLock_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RenewLock'
type LockManager_RenewLock_Call struct {
	*mock.Call
}

// RenewLock is a helper method to define mock.On call
//   - ctx context.Context
//   - lockTime time.Duration
//   - resource string
//   - token string
func (_e *LockManager_Expecter) RenewLock(ctx interface{}, lockTime interface{}, resource interface{}, token interface{}) *LockManager_RenewLock_Call {
	return &LockManager_RenewLock_Call{Call: _e.mock.On("RenewLock", ctx, lockTime, resource, token)}
}

func (_c *LockManager_RenewLock_Call) Run(run func(ctx context.Context, lockTime time.Duration, resource string, token string)) *LockManager_RenewLock_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(time.Duration), args[2].(string), args[3].(string))
	})
	return _c
}

func (_c *LockManager_RenewLock_Call) Return(_a0 error) *LockManager_RenewLock_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *LockManager_RenewLock_Call) RunAndReturn(run func(context.Context, time.Duration, string, string) error) *LockManager_RenewLock_Call {
	_c.Call.Return(run)
	return _c
}

// NewLockManager creates a new instance of LockManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewLockManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *LockManager {
	mock := &LockManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
