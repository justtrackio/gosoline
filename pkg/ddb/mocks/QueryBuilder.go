// Code generated by mockery v2.53.0. DO NOT EDIT.

package mocks

import (
	expression "github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	ddb "github.com/justtrackio/gosoline/pkg/ddb"
	mock "github.com/stretchr/testify/mock"
)

// QueryBuilder is an autogenerated mock type for the QueryBuilder type
type QueryBuilder struct {
	mock.Mock
}

type QueryBuilder_Expecter struct {
	mock *mock.Mock
}

func (_m *QueryBuilder) EXPECT() *QueryBuilder_Expecter {
	return &QueryBuilder_Expecter{mock: &_m.Mock}
}

// Build provides a mock function with given fields: result
func (_m *QueryBuilder) Build(result interface{}) (*ddb.QueryOperation, error) {
	ret := _m.Called(result)

	if len(ret) == 0 {
		panic("no return value specified for Build")
	}

	var r0 *ddb.QueryOperation
	var r1 error
	if rf, ok := ret.Get(0).(func(interface{}) (*ddb.QueryOperation, error)); ok {
		return rf(result)
	}
	if rf, ok := ret.Get(0).(func(interface{}) *ddb.QueryOperation); ok {
		r0 = rf(result)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*ddb.QueryOperation)
		}
	}

	if rf, ok := ret.Get(1).(func(interface{}) error); ok {
		r1 = rf(result)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryBuilder_Build_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Build'
type QueryBuilder_Build_Call struct {
	*mock.Call
}

// Build is a helper method to define mock.On call
//   - result interface{}
func (_e *QueryBuilder_Expecter) Build(result interface{}) *QueryBuilder_Build_Call {
	return &QueryBuilder_Build_Call{Call: _e.mock.On("Build", result)}
}

func (_c *QueryBuilder_Build_Call) Run(run func(result interface{})) *QueryBuilder_Build_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_Build_Call) Return(_a0 *ddb.QueryOperation, _a1 error) *QueryBuilder_Build_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *QueryBuilder_Build_Call) RunAndReturn(run func(interface{}) (*ddb.QueryOperation, error)) *QueryBuilder_Build_Call {
	_c.Call.Return(run)
	return _c
}

// DisableTtlFilter provides a mock function with no fields
func (_m *QueryBuilder) DisableTtlFilter() ddb.QueryBuilder {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for DisableTtlFilter")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func() ddb.QueryBuilder); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_DisableTtlFilter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'DisableTtlFilter'
type QueryBuilder_DisableTtlFilter_Call struct {
	*mock.Call
}

// DisableTtlFilter is a helper method to define mock.On call
func (_e *QueryBuilder_Expecter) DisableTtlFilter() *QueryBuilder_DisableTtlFilter_Call {
	return &QueryBuilder_DisableTtlFilter_Call{Call: _e.mock.On("DisableTtlFilter")}
}

func (_c *QueryBuilder_DisableTtlFilter_Call) Run(run func()) *QueryBuilder_DisableTtlFilter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *QueryBuilder_DisableTtlFilter_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_DisableTtlFilter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_DisableTtlFilter_Call) RunAndReturn(run func() ddb.QueryBuilder) *QueryBuilder_DisableTtlFilter_Call {
	_c.Call.Return(run)
	return _c
}

// WithConsistentRead provides a mock function with given fields: consistentRead
func (_m *QueryBuilder) WithConsistentRead(consistentRead bool) ddb.QueryBuilder {
	ret := _m.Called(consistentRead)

	if len(ret) == 0 {
		panic("no return value specified for WithConsistentRead")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(bool) ddb.QueryBuilder); ok {
		r0 = rf(consistentRead)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithConsistentRead_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithConsistentRead'
type QueryBuilder_WithConsistentRead_Call struct {
	*mock.Call
}

// WithConsistentRead is a helper method to define mock.On call
//   - consistentRead bool
func (_e *QueryBuilder_Expecter) WithConsistentRead(consistentRead interface{}) *QueryBuilder_WithConsistentRead_Call {
	return &QueryBuilder_WithConsistentRead_Call{Call: _e.mock.On("WithConsistentRead", consistentRead)}
}

func (_c *QueryBuilder_WithConsistentRead_Call) Run(run func(consistentRead bool)) *QueryBuilder_WithConsistentRead_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(bool))
	})
	return _c
}

func (_c *QueryBuilder_WithConsistentRead_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithConsistentRead_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithConsistentRead_Call) RunAndReturn(run func(bool) ddb.QueryBuilder) *QueryBuilder_WithConsistentRead_Call {
	_c.Call.Return(run)
	return _c
}

// WithDescendingOrder provides a mock function with no fields
func (_m *QueryBuilder) WithDescendingOrder() ddb.QueryBuilder {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for WithDescendingOrder")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func() ddb.QueryBuilder); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithDescendingOrder_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithDescendingOrder'
type QueryBuilder_WithDescendingOrder_Call struct {
	*mock.Call
}

// WithDescendingOrder is a helper method to define mock.On call
func (_e *QueryBuilder_Expecter) WithDescendingOrder() *QueryBuilder_WithDescendingOrder_Call {
	return &QueryBuilder_WithDescendingOrder_Call{Call: _e.mock.On("WithDescendingOrder")}
}

func (_c *QueryBuilder_WithDescendingOrder_Call) Run(run func()) *QueryBuilder_WithDescendingOrder_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *QueryBuilder_WithDescendingOrder_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithDescendingOrder_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithDescendingOrder_Call) RunAndReturn(run func() ddb.QueryBuilder) *QueryBuilder_WithDescendingOrder_Call {
	_c.Call.Return(run)
	return _c
}

// WithFilter provides a mock function with given fields: filter
func (_m *QueryBuilder) WithFilter(filter expression.ConditionBuilder) ddb.QueryBuilder {
	ret := _m.Called(filter)

	if len(ret) == 0 {
		panic("no return value specified for WithFilter")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(expression.ConditionBuilder) ddb.QueryBuilder); ok {
		r0 = rf(filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithFilter_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithFilter'
type QueryBuilder_WithFilter_Call struct {
	*mock.Call
}

// WithFilter is a helper method to define mock.On call
//   - filter expression.ConditionBuilder
func (_e *QueryBuilder_Expecter) WithFilter(filter interface{}) *QueryBuilder_WithFilter_Call {
	return &QueryBuilder_WithFilter_Call{Call: _e.mock.On("WithFilter", filter)}
}

func (_c *QueryBuilder_WithFilter_Call) Run(run func(filter expression.ConditionBuilder)) *QueryBuilder_WithFilter_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(expression.ConditionBuilder))
	})
	return _c
}

func (_c *QueryBuilder_WithFilter_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithFilter_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithFilter_Call) RunAndReturn(run func(expression.ConditionBuilder) ddb.QueryBuilder) *QueryBuilder_WithFilter_Call {
	_c.Call.Return(run)
	return _c
}

// WithHash provides a mock function with given fields: value
func (_m *QueryBuilder) WithHash(value interface{}) ddb.QueryBuilder {
	ret := _m.Called(value)

	if len(ret) == 0 {
		panic("no return value specified for WithHash")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}) ddb.QueryBuilder); ok {
		r0 = rf(value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithHash_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithHash'
type QueryBuilder_WithHash_Call struct {
	*mock.Call
}

// WithHash is a helper method to define mock.On call
//   - value interface{}
func (_e *QueryBuilder_Expecter) WithHash(value interface{}) *QueryBuilder_WithHash_Call {
	return &QueryBuilder_WithHash_Call{Call: _e.mock.On("WithHash", value)}
}

func (_c *QueryBuilder_WithHash_Call) Run(run func(value interface{})) *QueryBuilder_WithHash_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithHash_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithHash_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithHash_Call) RunAndReturn(run func(interface{}) ddb.QueryBuilder) *QueryBuilder_WithHash_Call {
	_c.Call.Return(run)
	return _c
}

// WithIndex provides a mock function with given fields: name
func (_m *QueryBuilder) WithIndex(name string) ddb.QueryBuilder {
	ret := _m.Called(name)

	if len(ret) == 0 {
		panic("no return value specified for WithIndex")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(string) ddb.QueryBuilder); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithIndex_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithIndex'
type QueryBuilder_WithIndex_Call struct {
	*mock.Call
}

// WithIndex is a helper method to define mock.On call
//   - name string
func (_e *QueryBuilder_Expecter) WithIndex(name interface{}) *QueryBuilder_WithIndex_Call {
	return &QueryBuilder_WithIndex_Call{Call: _e.mock.On("WithIndex", name)}
}

func (_c *QueryBuilder_WithIndex_Call) Run(run func(name string)) *QueryBuilder_WithIndex_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *QueryBuilder_WithIndex_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithIndex_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithIndex_Call) RunAndReturn(run func(string) ddb.QueryBuilder) *QueryBuilder_WithIndex_Call {
	_c.Call.Return(run)
	return _c
}

// WithLimit provides a mock function with given fields: limit
func (_m *QueryBuilder) WithLimit(limit int) ddb.QueryBuilder {
	ret := _m.Called(limit)

	if len(ret) == 0 {
		panic("no return value specified for WithLimit")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(int) ddb.QueryBuilder); ok {
		r0 = rf(limit)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithLimit_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithLimit'
type QueryBuilder_WithLimit_Call struct {
	*mock.Call
}

// WithLimit is a helper method to define mock.On call
//   - limit int
func (_e *QueryBuilder_Expecter) WithLimit(limit interface{}) *QueryBuilder_WithLimit_Call {
	return &QueryBuilder_WithLimit_Call{Call: _e.mock.On("WithLimit", limit)}
}

func (_c *QueryBuilder_WithLimit_Call) Run(run func(limit int)) *QueryBuilder_WithLimit_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *QueryBuilder_WithLimit_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithLimit_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithLimit_Call) RunAndReturn(run func(int) ddb.QueryBuilder) *QueryBuilder_WithLimit_Call {
	_c.Call.Return(run)
	return _c
}

// WithPageSize provides a mock function with given fields: size
func (_m *QueryBuilder) WithPageSize(size int) ddb.QueryBuilder {
	ret := _m.Called(size)

	if len(ret) == 0 {
		panic("no return value specified for WithPageSize")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(int) ddb.QueryBuilder); ok {
		r0 = rf(size)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithPageSize_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithPageSize'
type QueryBuilder_WithPageSize_Call struct {
	*mock.Call
}

// WithPageSize is a helper method to define mock.On call
//   - size int
func (_e *QueryBuilder_Expecter) WithPageSize(size interface{}) *QueryBuilder_WithPageSize_Call {
	return &QueryBuilder_WithPageSize_Call{Call: _e.mock.On("WithPageSize", size)}
}

func (_c *QueryBuilder_WithPageSize_Call) Run(run func(size int)) *QueryBuilder_WithPageSize_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(int))
	})
	return _c
}

func (_c *QueryBuilder_WithPageSize_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithPageSize_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithPageSize_Call) RunAndReturn(run func(int) ddb.QueryBuilder) *QueryBuilder_WithPageSize_Call {
	_c.Call.Return(run)
	return _c
}

// WithProjection provides a mock function with given fields: projection
func (_m *QueryBuilder) WithProjection(projection interface{}) ddb.QueryBuilder {
	ret := _m.Called(projection)

	if len(ret) == 0 {
		panic("no return value specified for WithProjection")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}) ddb.QueryBuilder); ok {
		r0 = rf(projection)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithProjection_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithProjection'
type QueryBuilder_WithProjection_Call struct {
	*mock.Call
}

// WithProjection is a helper method to define mock.On call
//   - projection interface{}
func (_e *QueryBuilder_Expecter) WithProjection(projection interface{}) *QueryBuilder_WithProjection_Call {
	return &QueryBuilder_WithProjection_Call{Call: _e.mock.On("WithProjection", projection)}
}

func (_c *QueryBuilder_WithProjection_Call) Run(run func(projection interface{})) *QueryBuilder_WithProjection_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithProjection_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithProjection_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithProjection_Call) RunAndReturn(run func(interface{}) ddb.QueryBuilder) *QueryBuilder_WithProjection_Call {
	_c.Call.Return(run)
	return _c
}

// WithRangeBeginsWith provides a mock function with given fields: prefix
func (_m *QueryBuilder) WithRangeBeginsWith(prefix string) ddb.QueryBuilder {
	ret := _m.Called(prefix)

	if len(ret) == 0 {
		panic("no return value specified for WithRangeBeginsWith")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(string) ddb.QueryBuilder); ok {
		r0 = rf(prefix)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithRangeBeginsWith_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithRangeBeginsWith'
type QueryBuilder_WithRangeBeginsWith_Call struct {
	*mock.Call
}

// WithRangeBeginsWith is a helper method to define mock.On call
//   - prefix string
func (_e *QueryBuilder_Expecter) WithRangeBeginsWith(prefix interface{}) *QueryBuilder_WithRangeBeginsWith_Call {
	return &QueryBuilder_WithRangeBeginsWith_Call{Call: _e.mock.On("WithRangeBeginsWith", prefix)}
}

func (_c *QueryBuilder_WithRangeBeginsWith_Call) Run(run func(prefix string)) *QueryBuilder_WithRangeBeginsWith_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *QueryBuilder_WithRangeBeginsWith_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithRangeBeginsWith_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithRangeBeginsWith_Call) RunAndReturn(run func(string) ddb.QueryBuilder) *QueryBuilder_WithRangeBeginsWith_Call {
	_c.Call.Return(run)
	return _c
}

// WithRangeBetween provides a mock function with given fields: lower, upper
func (_m *QueryBuilder) WithRangeBetween(lower interface{}, upper interface{}) ddb.QueryBuilder {
	ret := _m.Called(lower, upper)

	if len(ret) == 0 {
		panic("no return value specified for WithRangeBetween")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}, interface{}) ddb.QueryBuilder); ok {
		r0 = rf(lower, upper)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithRangeBetween_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithRangeBetween'
type QueryBuilder_WithRangeBetween_Call struct {
	*mock.Call
}

// WithRangeBetween is a helper method to define mock.On call
//   - lower interface{}
//   - upper interface{}
func (_e *QueryBuilder_Expecter) WithRangeBetween(lower interface{}, upper interface{}) *QueryBuilder_WithRangeBetween_Call {
	return &QueryBuilder_WithRangeBetween_Call{Call: _e.mock.On("WithRangeBetween", lower, upper)}
}

func (_c *QueryBuilder_WithRangeBetween_Call) Run(run func(lower interface{}, upper interface{})) *QueryBuilder_WithRangeBetween_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}), args[1].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithRangeBetween_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithRangeBetween_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithRangeBetween_Call) RunAndReturn(run func(interface{}, interface{}) ddb.QueryBuilder) *QueryBuilder_WithRangeBetween_Call {
	_c.Call.Return(run)
	return _c
}

// WithRangeEq provides a mock function with given fields: value
func (_m *QueryBuilder) WithRangeEq(value interface{}) ddb.QueryBuilder {
	ret := _m.Called(value)

	if len(ret) == 0 {
		panic("no return value specified for WithRangeEq")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}) ddb.QueryBuilder); ok {
		r0 = rf(value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithRangeEq_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithRangeEq'
type QueryBuilder_WithRangeEq_Call struct {
	*mock.Call
}

// WithRangeEq is a helper method to define mock.On call
//   - value interface{}
func (_e *QueryBuilder_Expecter) WithRangeEq(value interface{}) *QueryBuilder_WithRangeEq_Call {
	return &QueryBuilder_WithRangeEq_Call{Call: _e.mock.On("WithRangeEq", value)}
}

func (_c *QueryBuilder_WithRangeEq_Call) Run(run func(value interface{})) *QueryBuilder_WithRangeEq_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithRangeEq_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithRangeEq_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithRangeEq_Call) RunAndReturn(run func(interface{}) ddb.QueryBuilder) *QueryBuilder_WithRangeEq_Call {
	_c.Call.Return(run)
	return _c
}

// WithRangeGt provides a mock function with given fields: value
func (_m *QueryBuilder) WithRangeGt(value interface{}) ddb.QueryBuilder {
	ret := _m.Called(value)

	if len(ret) == 0 {
		panic("no return value specified for WithRangeGt")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}) ddb.QueryBuilder); ok {
		r0 = rf(value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithRangeGt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithRangeGt'
type QueryBuilder_WithRangeGt_Call struct {
	*mock.Call
}

// WithRangeGt is a helper method to define mock.On call
//   - value interface{}
func (_e *QueryBuilder_Expecter) WithRangeGt(value interface{}) *QueryBuilder_WithRangeGt_Call {
	return &QueryBuilder_WithRangeGt_Call{Call: _e.mock.On("WithRangeGt", value)}
}

func (_c *QueryBuilder_WithRangeGt_Call) Run(run func(value interface{})) *QueryBuilder_WithRangeGt_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithRangeGt_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithRangeGt_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithRangeGt_Call) RunAndReturn(run func(interface{}) ddb.QueryBuilder) *QueryBuilder_WithRangeGt_Call {
	_c.Call.Return(run)
	return _c
}

// WithRangeGte provides a mock function with given fields: value
func (_m *QueryBuilder) WithRangeGte(value interface{}) ddb.QueryBuilder {
	ret := _m.Called(value)

	if len(ret) == 0 {
		panic("no return value specified for WithRangeGte")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}) ddb.QueryBuilder); ok {
		r0 = rf(value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithRangeGte_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithRangeGte'
type QueryBuilder_WithRangeGte_Call struct {
	*mock.Call
}

// WithRangeGte is a helper method to define mock.On call
//   - value interface{}
func (_e *QueryBuilder_Expecter) WithRangeGte(value interface{}) *QueryBuilder_WithRangeGte_Call {
	return &QueryBuilder_WithRangeGte_Call{Call: _e.mock.On("WithRangeGte", value)}
}

func (_c *QueryBuilder_WithRangeGte_Call) Run(run func(value interface{})) *QueryBuilder_WithRangeGte_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithRangeGte_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithRangeGte_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithRangeGte_Call) RunAndReturn(run func(interface{}) ddb.QueryBuilder) *QueryBuilder_WithRangeGte_Call {
	_c.Call.Return(run)
	return _c
}

// WithRangeLt provides a mock function with given fields: value
func (_m *QueryBuilder) WithRangeLt(value interface{}) ddb.QueryBuilder {
	ret := _m.Called(value)

	if len(ret) == 0 {
		panic("no return value specified for WithRangeLt")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}) ddb.QueryBuilder); ok {
		r0 = rf(value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithRangeLt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithRangeLt'
type QueryBuilder_WithRangeLt_Call struct {
	*mock.Call
}

// WithRangeLt is a helper method to define mock.On call
//   - value interface{}
func (_e *QueryBuilder_Expecter) WithRangeLt(value interface{}) *QueryBuilder_WithRangeLt_Call {
	return &QueryBuilder_WithRangeLt_Call{Call: _e.mock.On("WithRangeLt", value)}
}

func (_c *QueryBuilder_WithRangeLt_Call) Run(run func(value interface{})) *QueryBuilder_WithRangeLt_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithRangeLt_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithRangeLt_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithRangeLt_Call) RunAndReturn(run func(interface{}) ddb.QueryBuilder) *QueryBuilder_WithRangeLt_Call {
	_c.Call.Return(run)
	return _c
}

// WithRangeLte provides a mock function with given fields: value
func (_m *QueryBuilder) WithRangeLte(value interface{}) ddb.QueryBuilder {
	ret := _m.Called(value)

	if len(ret) == 0 {
		panic("no return value specified for WithRangeLte")
	}

	var r0 ddb.QueryBuilder
	if rf, ok := ret.Get(0).(func(interface{}) ddb.QueryBuilder); ok {
		r0 = rf(value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ddb.QueryBuilder)
		}
	}

	return r0
}

// QueryBuilder_WithRangeLte_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'WithRangeLte'
type QueryBuilder_WithRangeLte_Call struct {
	*mock.Call
}

// WithRangeLte is a helper method to define mock.On call
//   - value interface{}
func (_e *QueryBuilder_Expecter) WithRangeLte(value interface{}) *QueryBuilder_WithRangeLte_Call {
	return &QueryBuilder_WithRangeLte_Call{Call: _e.mock.On("WithRangeLte", value)}
}

func (_c *QueryBuilder_WithRangeLte_Call) Run(run func(value interface{})) *QueryBuilder_WithRangeLte_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *QueryBuilder_WithRangeLte_Call) Return(_a0 ddb.QueryBuilder) *QueryBuilder_WithRangeLte_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *QueryBuilder_WithRangeLte_Call) RunAndReturn(run func(interface{}) ddb.QueryBuilder) *QueryBuilder_WithRangeLte_Call {
	_c.Call.Return(run)
	return _c
}

// NewQueryBuilder creates a new instance of QueryBuilder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewQueryBuilder(t interface {
	mock.TestingT
	Cleanup(func())
}) *QueryBuilder {
	mock := &QueryBuilder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
