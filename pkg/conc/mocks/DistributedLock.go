// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"
import time "time"

// DistributedLock is an autogenerated mock type for the DistributedLock type
type DistributedLock struct {
	mock.Mock
}

// Release provides a mock function with given fields:
func (_m *DistributedLock) Release() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Renew provides a mock function with given fields: ctx, lockTime
func (_m *DistributedLock) Renew(ctx context.Context, lockTime time.Duration) error {
	ret := _m.Called(ctx, lockTime)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Duration) error); ok {
		r0 = rf(ctx, lockTime)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}