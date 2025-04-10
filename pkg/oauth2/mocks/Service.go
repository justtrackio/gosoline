// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	oauth2 "github.com/justtrackio/gosoline/pkg/oauth2"
	mock "github.com/stretchr/testify/mock"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

type Service_Expecter struct {
	mock *mock.Mock
}

func (_m *Service) EXPECT() *Service_Expecter {
	return &Service_Expecter{mock: &_m.Mock}
}

// GetAuthRefresh provides a mock function with given fields: ctx, authRequest
func (_m *Service) GetAuthRefresh(ctx context.Context, authRequest *oauth2.GoogleAuthRequest) (*oauth2.GoogleAuthResponse, error) {
	ret := _m.Called(ctx, authRequest)

	if len(ret) == 0 {
		panic("no return value specified for GetAuthRefresh")
	}

	var r0 *oauth2.GoogleAuthResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *oauth2.GoogleAuthRequest) (*oauth2.GoogleAuthResponse, error)); ok {
		return rf(ctx, authRequest)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *oauth2.GoogleAuthRequest) *oauth2.GoogleAuthResponse); ok {
		r0 = rf(ctx, authRequest)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*oauth2.GoogleAuthResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *oauth2.GoogleAuthRequest) error); ok {
		r1 = rf(ctx, authRequest)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Service_GetAuthRefresh_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetAuthRefresh'
type Service_GetAuthRefresh_Call struct {
	*mock.Call
}

// GetAuthRefresh is a helper method to define mock.On call
//   - ctx context.Context
//   - authRequest *oauth2.GoogleAuthRequest
func (_e *Service_Expecter) GetAuthRefresh(ctx interface{}, authRequest interface{}) *Service_GetAuthRefresh_Call {
	return &Service_GetAuthRefresh_Call{Call: _e.mock.On("GetAuthRefresh", ctx, authRequest)}
}

func (_c *Service_GetAuthRefresh_Call) Run(run func(ctx context.Context, authRequest *oauth2.GoogleAuthRequest)) *Service_GetAuthRefresh_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*oauth2.GoogleAuthRequest))
	})
	return _c
}

func (_c *Service_GetAuthRefresh_Call) Return(_a0 *oauth2.GoogleAuthResponse, _a1 error) *Service_GetAuthRefresh_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Service_GetAuthRefresh_Call) RunAndReturn(run func(context.Context, *oauth2.GoogleAuthRequest) (*oauth2.GoogleAuthResponse, error)) *Service_GetAuthRefresh_Call {
	_c.Call.Return(run)
	return _c
}

// TokenInfo provides a mock function with given fields: ctx, accessToken
func (_m *Service) TokenInfo(ctx context.Context, accessToken string) (*oauth2.GoogleTokenInfoResponse, error) {
	ret := _m.Called(ctx, accessToken)

	if len(ret) == 0 {
		panic("no return value specified for TokenInfo")
	}

	var r0 *oauth2.GoogleTokenInfoResponse
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*oauth2.GoogleTokenInfoResponse, error)); ok {
		return rf(ctx, accessToken)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *oauth2.GoogleTokenInfoResponse); ok {
		r0 = rf(ctx, accessToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*oauth2.GoogleTokenInfoResponse)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, accessToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Service_TokenInfo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'TokenInfo'
type Service_TokenInfo_Call struct {
	*mock.Call
}

// TokenInfo is a helper method to define mock.On call
//   - ctx context.Context
//   - accessToken string
func (_e *Service_Expecter) TokenInfo(ctx interface{}, accessToken interface{}) *Service_TokenInfo_Call {
	return &Service_TokenInfo_Call{Call: _e.mock.On("TokenInfo", ctx, accessToken)}
}

func (_c *Service_TokenInfo_Call) Run(run func(ctx context.Context, accessToken string)) *Service_TokenInfo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Service_TokenInfo_Call) Return(_a0 *oauth2.GoogleTokenInfoResponse, _a1 error) *Service_TokenInfo_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Service_TokenInfo_Call) RunAndReturn(run func(context.Context, string) (*oauth2.GoogleTokenInfoResponse, error)) *Service_TokenInfo_Call {
	_c.Call.Return(run)
	return _c
}

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewService(t interface {
	mock.TestingT
	Cleanup(func())
}) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
