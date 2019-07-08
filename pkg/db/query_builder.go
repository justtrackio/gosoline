package db

import "github.com/Masterminds/squirrel"

//go:generate mockery -name QueryBuilder
type QueryBuilder interface {
	Table(table string) QueryBuilder
	Joins(joins []string) QueryBuilder
	Where(query interface{}, args ...interface{}) QueryBuilder
	GroupBy(field ...string) QueryBuilder
	OrderBy(field string, direction string) QueryBuilder
	Page(offset int, size int) QueryBuilder
}

type RawQueryBuilder struct {
	Builder squirrel.SelectBuilder
}

func NewQueryBuilder() *RawQueryBuilder {
	return &RawQueryBuilder{
		Builder: squirrel.SelectBuilder{}.PlaceholderFormat(squirrel.Question),
	}
}

func (b *RawQueryBuilder) Table(table string) QueryBuilder {
	b.Builder = b.Builder.From(table)

	return b
}

func (b *RawQueryBuilder) Joins(joins []string) QueryBuilder {
	for _, join := range joins {
		b.Builder = b.Builder.JoinClause(join)
	}

	return b
}

func (b *RawQueryBuilder) Where(query interface{}, args ...interface{}) QueryBuilder {
	b.Builder = b.Builder.Where(query, args...)

	return b
}

func (b *RawQueryBuilder) GroupBy(field ...string) QueryBuilder {
	b.Builder = b.Builder.GroupBy(field...)

	return b
}

func (b *RawQueryBuilder) OrderBy(field string, direction string) QueryBuilder {
	b.Builder = b.Builder.OrderBy(field, direction)

	return b
}

func (b *RawQueryBuilder) Page(offset int, size int) QueryBuilder {
	b.Builder = b.Builder.Offset(uint64(offset)).Limit(uint64(size))

	return b
}
