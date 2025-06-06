// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	context "context"

	kinesis "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	mock "github.com/stretchr/testify/mock"
)

// Checkpoint is an autogenerated mock type for the Checkpoint type
type Checkpoint struct {
	mock.Mock
}

type Checkpoint_Expecter struct {
	mock *mock.Mock
}

func (_m *Checkpoint) EXPECT() *Checkpoint_Expecter {
	return &Checkpoint_Expecter{mock: &_m.Mock}
}

// Advance provides a mock function with given fields: sequenceNumber, shardIterator
func (_m *Checkpoint) Advance(sequenceNumber kinesis.SequenceNumber, shardIterator kinesis.ShardIterator) error {
	ret := _m.Called(sequenceNumber, shardIterator)

	if len(ret) == 0 {
		panic("no return value specified for Advance")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(kinesis.SequenceNumber, kinesis.ShardIterator) error); ok {
		r0 = rf(sequenceNumber, shardIterator)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Checkpoint_Advance_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Advance'
type Checkpoint_Advance_Call struct {
	*mock.Call
}

// Advance is a helper method to define mock.On call
//   - sequenceNumber kinesis.SequenceNumber
//   - shardIterator kinesis.ShardIterator
func (_e *Checkpoint_Expecter) Advance(sequenceNumber interface{}, shardIterator interface{}) *Checkpoint_Advance_Call {
	return &Checkpoint_Advance_Call{Call: _e.mock.On("Advance", sequenceNumber, shardIterator)}
}

func (_c *Checkpoint_Advance_Call) Run(run func(sequenceNumber kinesis.SequenceNumber, shardIterator kinesis.ShardIterator)) *Checkpoint_Advance_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(kinesis.SequenceNumber), args[1].(kinesis.ShardIterator))
	})
	return _c
}

func (_c *Checkpoint_Advance_Call) Return(_a0 error) *Checkpoint_Advance_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checkpoint_Advance_Call) RunAndReturn(run func(kinesis.SequenceNumber, kinesis.ShardIterator) error) *Checkpoint_Advance_Call {
	_c.Call.Return(run)
	return _c
}

// Done provides a mock function with given fields: sequenceNumber
func (_m *Checkpoint) Done(sequenceNumber kinesis.SequenceNumber) error {
	ret := _m.Called(sequenceNumber)

	if len(ret) == 0 {
		panic("no return value specified for Done")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(kinesis.SequenceNumber) error); ok {
		r0 = rf(sequenceNumber)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Checkpoint_Done_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Done'
type Checkpoint_Done_Call struct {
	*mock.Call
}

// Done is a helper method to define mock.On call
//   - sequenceNumber kinesis.SequenceNumber
func (_e *Checkpoint_Expecter) Done(sequenceNumber interface{}) *Checkpoint_Done_Call {
	return &Checkpoint_Done_Call{Call: _e.mock.On("Done", sequenceNumber)}
}

func (_c *Checkpoint_Done_Call) Run(run func(sequenceNumber kinesis.SequenceNumber)) *Checkpoint_Done_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(kinesis.SequenceNumber))
	})
	return _c
}

func (_c *Checkpoint_Done_Call) Return(_a0 error) *Checkpoint_Done_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checkpoint_Done_Call) RunAndReturn(run func(kinesis.SequenceNumber) error) *Checkpoint_Done_Call {
	_c.Call.Return(run)
	return _c
}

// GetSequenceNumber provides a mock function with no fields
func (_m *Checkpoint) GetSequenceNumber() kinesis.SequenceNumber {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetSequenceNumber")
	}

	var r0 kinesis.SequenceNumber
	if rf, ok := ret.Get(0).(func() kinesis.SequenceNumber); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(kinesis.SequenceNumber)
	}

	return r0
}

// Checkpoint_GetSequenceNumber_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetSequenceNumber'
type Checkpoint_GetSequenceNumber_Call struct {
	*mock.Call
}

// GetSequenceNumber is a helper method to define mock.On call
func (_e *Checkpoint_Expecter) GetSequenceNumber() *Checkpoint_GetSequenceNumber_Call {
	return &Checkpoint_GetSequenceNumber_Call{Call: _e.mock.On("GetSequenceNumber")}
}

func (_c *Checkpoint_GetSequenceNumber_Call) Run(run func()) *Checkpoint_GetSequenceNumber_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Checkpoint_GetSequenceNumber_Call) Return(_a0 kinesis.SequenceNumber) *Checkpoint_GetSequenceNumber_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checkpoint_GetSequenceNumber_Call) RunAndReturn(run func() kinesis.SequenceNumber) *Checkpoint_GetSequenceNumber_Call {
	_c.Call.Return(run)
	return _c
}

// GetShardIterator provides a mock function with no fields
func (_m *Checkpoint) GetShardIterator() kinesis.ShardIterator {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetShardIterator")
	}

	var r0 kinesis.ShardIterator
	if rf, ok := ret.Get(0).(func() kinesis.ShardIterator); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(kinesis.ShardIterator)
	}

	return r0
}

// Checkpoint_GetShardIterator_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetShardIterator'
type Checkpoint_GetShardIterator_Call struct {
	*mock.Call
}

// GetShardIterator is a helper method to define mock.On call
func (_e *Checkpoint_Expecter) GetShardIterator() *Checkpoint_GetShardIterator_Call {
	return &Checkpoint_GetShardIterator_Call{Call: _e.mock.On("GetShardIterator")}
}

func (_c *Checkpoint_GetShardIterator_Call) Run(run func()) *Checkpoint_GetShardIterator_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Checkpoint_GetShardIterator_Call) Return(_a0 kinesis.ShardIterator) *Checkpoint_GetShardIterator_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checkpoint_GetShardIterator_Call) RunAndReturn(run func() kinesis.ShardIterator) *Checkpoint_GetShardIterator_Call {
	_c.Call.Return(run)
	return _c
}

// Persist provides a mock function with given fields: ctx
func (_m *Checkpoint) Persist(ctx context.Context) (bool, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Persist")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (bool, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Checkpoint_Persist_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Persist'
type Checkpoint_Persist_Call struct {
	*mock.Call
}

// Persist is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Checkpoint_Expecter) Persist(ctx interface{}) *Checkpoint_Persist_Call {
	return &Checkpoint_Persist_Call{Call: _e.mock.On("Persist", ctx)}
}

func (_c *Checkpoint_Persist_Call) Run(run func(ctx context.Context)) *Checkpoint_Persist_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Checkpoint_Persist_Call) Return(shouldRelease bool, err error) *Checkpoint_Persist_Call {
	_c.Call.Return(shouldRelease, err)
	return _c
}

func (_c *Checkpoint_Persist_Call) RunAndReturn(run func(context.Context) (bool, error)) *Checkpoint_Persist_Call {
	_c.Call.Return(run)
	return _c
}

// Release provides a mock function with given fields: ctx
func (_m *Checkpoint) Release(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Release")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Checkpoint_Release_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Release'
type Checkpoint_Release_Call struct {
	*mock.Call
}

// Release is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Checkpoint_Expecter) Release(ctx interface{}) *Checkpoint_Release_Call {
	return &Checkpoint_Release_Call{Call: _e.mock.On("Release", ctx)}
}

func (_c *Checkpoint_Release_Call) Run(run func(ctx context.Context)) *Checkpoint_Release_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *Checkpoint_Release_Call) Return(_a0 error) *Checkpoint_Release_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Checkpoint_Release_Call) RunAndReturn(run func(context.Context) error) *Checkpoint_Release_Call {
	_c.Call.Return(run)
	return _c
}

// NewCheckpoint creates a new instance of Checkpoint. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewCheckpoint(t interface {
	mock.TestingT
	Cleanup(func())
}) *Checkpoint {
	mock := &Checkpoint{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
