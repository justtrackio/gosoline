// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mailpit "github.com/justtrackio/gosoline/pkg/test/env/mailpit"
	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

type Client_Expecter struct {
	mock *mock.Mock
}

func (_m *Client) EXPECT() *Client_Expecter {
	return &Client_Expecter{mock: &_m.Mock}
}

// GetMessage provides a mock function with given fields: ctx, id
func (_m *Client) GetMessage(ctx context.Context, id string) (*mailpit.Message, error) {
	ret := _m.Called(ctx, id)

	if len(ret) == 0 {
		panic("no return value specified for GetMessage")
	}

	var r0 *mailpit.Message
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*mailpit.Message, error)); ok {
		return rf(ctx, id)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *mailpit.Message); ok {
		r0 = rf(ctx, id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*mailpit.Message)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_GetMessage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetMessage'
type Client_GetMessage_Call struct {
	*mock.Call
}

// GetMessage is a helper method to define mock.On call
//   - ctx context.Context
//   - id string
func (_e *Client_Expecter) GetMessage(ctx interface{}, id interface{}) *Client_GetMessage_Call {
	return &Client_GetMessage_Call{Call: _e.mock.On("GetMessage", ctx, id)}
}

func (_c *Client_GetMessage_Call) Run(run func(ctx context.Context, id string)) *Client_GetMessage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Client_GetMessage_Call) Return(_a0 *mailpit.Message, _a1 error) *Client_GetMessage_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_GetMessage_Call) RunAndReturn(run func(context.Context, string) (*mailpit.Message, error)) *Client_GetMessage_Call {
	_c.Call.Return(run)
	return _c
}

// ListMessages provides a mock function with given fields: ctx
func (_m *Client) ListMessages(ctx context.Context) (*mailpit.ListMessagesResponse, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for ListMessages")
	}

	var r0 *mailpit.ListMessagesResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*mailpit.ListMessagesResponse, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *mailpit.ListMessagesResponse); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*mailpit.ListMessagesResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Client_ListMessages_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ListMessages'
type Client_ListMessages_Call struct {
	*mock.Call
}

// ListMessages is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Client_Expecter) ListMessages(ctx interface{}) *Client_ListMessages_Call {
	return &Client_ListMessages_Call{Call: _e.mock.On("ListMessages", ctx)}
}

func (_c *Client_ListMessages_Call) Run(run func(ctx context.Context)) *Client_ListMessages_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Client_ListMessages_Call) Return(_a0 *mailpit.ListMessagesResponse, _a1 error) *Client_ListMessages_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Client_ListMessages_Call) RunAndReturn(run func(context.Context) (*mailpit.ListMessagesResponse, error)) *Client_ListMessages_Call {
	_c.Call.Return(run)
	return _c
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
