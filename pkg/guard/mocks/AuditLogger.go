// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	ladon "github.com/selm0/ladon"

	mock "github.com/stretchr/testify/mock"
)

// AuditLogger is an autogenerated mock type for the AuditLogger type
type AuditLogger struct {
	mock.Mock
}

type AuditLogger_Expecter struct {
	mock *mock.Mock
}

func (_m *AuditLogger) EXPECT() *AuditLogger_Expecter {
	return &AuditLogger_Expecter{mock: &_m.Mock}
}

// LogGrantedAccessRequest provides a mock function with given fields: ctx, request, pool, deciders
func (_m *AuditLogger) LogGrantedAccessRequest(ctx context.Context, request *ladon.Request, pool ladon.Policies, deciders ladon.Policies) {
	_m.Called(ctx, request, pool, deciders)
}

// AuditLogger_LogGrantedAccessRequest_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'LogGrantedAccessRequest'
type AuditLogger_LogGrantedAccessRequest_Call struct {
	*mock.Call
}

// LogGrantedAccessRequest is a helper method to define mock.On call
//   - ctx context.Context
//   - request *ladon.Request
//   - pool ladon.Policies
//   - deciders ladon.Policies
func (_e *AuditLogger_Expecter) LogGrantedAccessRequest(ctx interface{}, request interface{}, pool interface{}, deciders interface{}) *AuditLogger_LogGrantedAccessRequest_Call {
	return &AuditLogger_LogGrantedAccessRequest_Call{Call: _e.mock.On("LogGrantedAccessRequest", ctx, request, pool, deciders)}
}

func (_c *AuditLogger_LogGrantedAccessRequest_Call) Run(run func(ctx context.Context, request *ladon.Request, pool ladon.Policies, deciders ladon.Policies)) *AuditLogger_LogGrantedAccessRequest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*ladon.Request), args[2].(ladon.Policies), args[3].(ladon.Policies))
	})
	return _c
}

func (_c *AuditLogger_LogGrantedAccessRequest_Call) Return() *AuditLogger_LogGrantedAccessRequest_Call {
	_c.Call.Return()
	return _c
}

func (_c *AuditLogger_LogGrantedAccessRequest_Call) RunAndReturn(run func(context.Context, *ladon.Request, ladon.Policies, ladon.Policies)) *AuditLogger_LogGrantedAccessRequest_Call {
	_c.Run(run)
	return _c
}

// LogRejectedAccessRequest provides a mock function with given fields: ctx, request, pool, deciders
func (_m *AuditLogger) LogRejectedAccessRequest(ctx context.Context, request *ladon.Request, pool ladon.Policies, deciders ladon.Policies) {
	_m.Called(ctx, request, pool, deciders)
}

// AuditLogger_LogRejectedAccessRequest_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'LogRejectedAccessRequest'
type AuditLogger_LogRejectedAccessRequest_Call struct {
	*mock.Call
}

// LogRejectedAccessRequest is a helper method to define mock.On call
//   - ctx context.Context
//   - request *ladon.Request
//   - pool ladon.Policies
//   - deciders ladon.Policies
func (_e *AuditLogger_Expecter) LogRejectedAccessRequest(ctx interface{}, request interface{}, pool interface{}, deciders interface{}) *AuditLogger_LogRejectedAccessRequest_Call {
	return &AuditLogger_LogRejectedAccessRequest_Call{Call: _e.mock.On("LogRejectedAccessRequest", ctx, request, pool, deciders)}
}

func (_c *AuditLogger_LogRejectedAccessRequest_Call) Run(run func(ctx context.Context, request *ladon.Request, pool ladon.Policies, deciders ladon.Policies)) *AuditLogger_LogRejectedAccessRequest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*ladon.Request), args[2].(ladon.Policies), args[3].(ladon.Policies))
	})
	return _c
}

func (_c *AuditLogger_LogRejectedAccessRequest_Call) Return() *AuditLogger_LogRejectedAccessRequest_Call {
	_c.Call.Return()
	return _c
}

func (_c *AuditLogger_LogRejectedAccessRequest_Call) RunAndReturn(run func(context.Context, *ladon.Request, ladon.Policies, ladon.Policies)) *AuditLogger_LogRejectedAccessRequest_Call {
	_c.Run(run)
	return _c
}

// NewAuditLogger creates a new instance of AuditLogger. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAuditLogger(t interface {
	mock.TestingT
	Cleanup(func())
}) *AuditLogger {
	mock := &AuditLogger{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
