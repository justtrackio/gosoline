package ddb

import "github.com/adjoeio/djoemo"

//go:generate mockery -name QueryBuilder
type QueryBuilder interface {
	Index() *string
	WithHash(key string, value interface{}) QueryBuilder
	WithRange(key string, op djoemo.Operator, value interface{}) QueryBuilder
	WithIndex(name string) QueryBuilder
	Build() djoemo.QueryInterface
}

type queryBuilder struct {
	query
	index *string
}

func NewQueryBuilder(tableName string) QueryBuilder {
	return &queryBuilder{
		query: query{
			tableName: tableName,
		},
	}
}

func (qb *queryBuilder) Index() *string {
	return qb.index
}

func (qb *queryBuilder) WithHash(key string, value interface{}) QueryBuilder {
	qb.hashKeyName = key
	qb.hashKey = value

	return qb
}

func (qb *queryBuilder) WithIndex(name string) QueryBuilder {
	qb.index = &name

	return qb
}

func (qb *queryBuilder) WithRange(key string, op djoemo.Operator, value interface{}) QueryBuilder {
	qb.rangeKeyName = key
	qb.rangeOp = op
	qb.rangeKey = value

	return qb
}

func (qb *queryBuilder) Build() djoemo.QueryInterface {
	return &qb.query
}
