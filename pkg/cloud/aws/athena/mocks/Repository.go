// Code generated by mockery v2.46.0. DO NOT EDIT.

package mocks

import (
	context "context"

	sqlx "github.com/jmoiron/sqlx"
	mock "github.com/stretchr/testify/mock"

	squirrel "github.com/Masterminds/squirrel"
)

// Repository is an autogenerated mock type for the Repository type
type Repository[T interface{}] struct {
	mock.Mock
}

type Repository_Expecter[T interface{}] struct {
	mock *mock.Mock
}

func (_m *Repository[T]) EXPECT() *Repository_Expecter[T] {
	return &Repository_Expecter[T]{mock: &_m.Mock}
}

// Query provides a mock function with given fields: ctx, qb
func (_m *Repository[T]) Query(ctx context.Context, qb squirrel.SelectBuilder) ([]T, error) {
	ret := _m.Called(ctx, qb)

	if len(ret) == 0 {
		panic("no return value specified for Query")
	}

	var r0 []T
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, squirrel.SelectBuilder) ([]T, error)); ok {
		return rf(ctx, qb)
	}
	if rf, ok := ret.Get(0).(func(context.Context, squirrel.SelectBuilder) []T); ok {
		r0 = rf(ctx, qb)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]T)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, squirrel.SelectBuilder) error); ok {
		r1 = rf(ctx, qb)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Repository_Query_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Query'
type Repository_Query_Call[T interface{}] struct {
	*mock.Call
}

// Query is a helper method to define mock.On call
//   - ctx context.Context
//   - qb squirrel.SelectBuilder
func (_e *Repository_Expecter[T]) Query(ctx interface{}, qb interface{}) *Repository_Query_Call[T] {
	return &Repository_Query_Call[T]{Call: _e.mock.On("Query", ctx, qb)}
}

func (_c *Repository_Query_Call[T]) Run(run func(ctx context.Context, qb squirrel.SelectBuilder)) *Repository_Query_Call[T] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(squirrel.SelectBuilder))
	})
	return _c
}

func (_c *Repository_Query_Call[T]) Return(_a0 []T, _a1 error) *Repository_Query_Call[T] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Repository_Query_Call[T]) RunAndReturn(run func(context.Context, squirrel.SelectBuilder) ([]T, error)) *Repository_Query_Call[T] {
	_c.Call.Return(run)
	return _c
}

// QueryBuilder provides a mock function with given fields:
func (_m *Repository[T]) QueryBuilder() squirrel.SelectBuilder {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for QueryBuilder")
	}

	var r0 squirrel.SelectBuilder
	if rf, ok := ret.Get(0).(func() squirrel.SelectBuilder); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(squirrel.SelectBuilder)
	}

	return r0
}

// Repository_QueryBuilder_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'QueryBuilder'
type Repository_QueryBuilder_Call[T interface{}] struct {
	*mock.Call
}

// QueryBuilder is a helper method to define mock.On call
func (_e *Repository_Expecter[T]) QueryBuilder() *Repository_QueryBuilder_Call[T] {
	return &Repository_QueryBuilder_Call[T]{Call: _e.mock.On("QueryBuilder")}
}

func (_c *Repository_QueryBuilder_Call[T]) Run(run func()) *Repository_QueryBuilder_Call[T] {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Repository_QueryBuilder_Call[T]) Return(_a0 squirrel.SelectBuilder) *Repository_QueryBuilder_Call[T] {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Repository_QueryBuilder_Call[T]) RunAndReturn(run func() squirrel.SelectBuilder) *Repository_QueryBuilder_Call[T] {
	_c.Call.Return(run)
	return _c
}

// QueryRows provides a mock function with given fields: ctx, sql
func (_m *Repository[T]) QueryRows(ctx context.Context, sql string) (*sqlx.Rows, error) {
	ret := _m.Called(ctx, sql)

	if len(ret) == 0 {
		panic("no return value specified for QueryRows")
	}

	var r0 *sqlx.Rows
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*sqlx.Rows, error)); ok {
		return rf(ctx, sql)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *sqlx.Rows); ok {
		r0 = rf(ctx, sql)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sqlx.Rows)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, sql)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Repository_QueryRows_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'QueryRows'
type Repository_QueryRows_Call[T interface{}] struct {
	*mock.Call
}

// QueryRows is a helper method to define mock.On call
//   - ctx context.Context
//   - sql string
func (_e *Repository_Expecter[T]) QueryRows(ctx interface{}, sql interface{}) *Repository_QueryRows_Call[T] {
	return &Repository_QueryRows_Call[T]{Call: _e.mock.On("QueryRows", ctx, sql)}
}

func (_c *Repository_QueryRows_Call[T]) Run(run func(ctx context.Context, sql string)) *Repository_QueryRows_Call[T] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Repository_QueryRows_Call[T]) Return(_a0 *sqlx.Rows, _a1 error) *Repository_QueryRows_Call[T] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Repository_QueryRows_Call[T]) RunAndReturn(run func(context.Context, string) (*sqlx.Rows, error)) *Repository_QueryRows_Call[T] {
	_c.Call.Return(run)
	return _c
}

// QuerySql provides a mock function with given fields: ctx, query
func (_m *Repository[T]) QuerySql(ctx context.Context, query string) ([]T, error) {
	ret := _m.Called(ctx, query)

	if len(ret) == 0 {
		panic("no return value specified for QuerySql")
	}

	var r0 []T
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]T, error)); ok {
		return rf(ctx, query)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []T); ok {
		r0 = rf(ctx, query)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]T)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, query)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Repository_QuerySql_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'QuerySql'
type Repository_QuerySql_Call[T interface{}] struct {
	*mock.Call
}

// QuerySql is a helper method to define mock.On call
//   - ctx context.Context
//   - query string
func (_e *Repository_Expecter[T]) QuerySql(ctx interface{}, query interface{}) *Repository_QuerySql_Call[T] {
	return &Repository_QuerySql_Call[T]{Call: _e.mock.On("QuerySql", ctx, query)}
}

func (_c *Repository_QuerySql_Call[T]) Run(run func(ctx context.Context, query string)) *Repository_QuerySql_Call[T] {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Repository_QuerySql_Call[T]) Return(_a0 []T, _a1 error) *Repository_QuerySql_Call[T] {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Repository_QuerySql_Call[T]) RunAndReturn(run func(context.Context, string) ([]T, error)) *Repository_QuerySql_Call[T] {
	_c.Call.Return(run)
	return _c
}

// NewRepository creates a new instance of Repository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRepository[T interface{}](t interface {
	mock.TestingT
	Cleanup(func())
}) *Repository[T] {
	mock := &Repository[T]{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}