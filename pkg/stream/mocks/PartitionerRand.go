// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// PartitionerRand is an autogenerated mock type for the PartitionerRand type
type PartitionerRand struct {
	mock.Mock
}

type PartitionerRand_Expecter struct {
	mock *mock.Mock
}

func (_m *PartitionerRand) EXPECT() *PartitionerRand_Expecter {
	return &PartitionerRand_Expecter{mock: &_m.Mock}
}

// Intn provides a mock function with given fields: n
func (_m *PartitionerRand) Intn(n int) int {
	ret := _m.Called(n)

	if len(ret) == 0 {
		panic("no return value specified for Intn")
	}

	var r0 int
	if rf, ok := ret.Get(0).(func(int) int); ok {
		r0 = rf(n)
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// PartitionerRand_Intn_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Intn'
type PartitionerRand_Intn_Call struct {
	*mock.Call
}

// Intn is a helper method to define mock.On call
//   - n int
func (_e *PartitionerRand_Expecter) Intn(n interface{}) *PartitionerRand_Intn_Call {
	return &PartitionerRand_Intn_Call{Call: _e.mock.On("Intn", n)}
}

func (_c *PartitionerRand_Intn_Call) Run(run func(n int)) *PartitionerRand_Intn_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *PartitionerRand_Intn_Call) Return(_a0 int) *PartitionerRand_Intn_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *PartitionerRand_Intn_Call) RunAndReturn(run func(int) int) *PartitionerRand_Intn_Call {
	_c.Call.Return(run)
	return _c
}

// NewPartitionerRand creates a new instance of PartitionerRand. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPartitionerRand(t interface {
	mock.TestingT
	Cleanup(func())
}) *PartitionerRand {
	mock := &PartitionerRand{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
