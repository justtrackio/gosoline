package db_repo

import (
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/funk"
)

type page struct {
	offset int
	limit  int
}

type order struct {
	field     string
	direction any
}

type QueryBuilder struct {
	table   string
	joins   []string
	where   []any
	args    [][]any
	groupBy []string
	orderBy []order
	page    *page
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		joins:   make([]string, 0),
		groupBy: make([]string, 0),
		orderBy: make([]order, 0),
	}
}

func (qb *QueryBuilder) Table(table string) db.QueryBuilder {
	qb.table = table

	return qb
}

func (qb *QueryBuilder) Joins(joins []string) db.QueryBuilder {
	qb.joins = funk.Uniq(joins)

	return qb
}

func (qb *QueryBuilder) Where(query any, args ...any) db.QueryBuilder {
	qb.where = append(qb.where, query)
	qb.args = append(qb.args, args)

	return qb
}

func (qb *QueryBuilder) GroupBy(field ...string) db.QueryBuilder {
	qb.groupBy = append(qb.groupBy, field...)

	return qb
}

func (qb *QueryBuilder) OrderBy(field string, direction string) db.QueryBuilder {
	order := order{
		field:     field,
		direction: direction,
	}

	qb.orderBy = append(qb.orderBy, order)

	return qb
}

func (qb *QueryBuilder) Page(offset int, size int) db.QueryBuilder {
	qb.page = &page{
		offset: offset,
		limit:  size,
	}

	return qb
}
