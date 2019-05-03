package db_repo

import "github.com/thoas/go-funk"

type page struct {
	offset int
	limit  int
}

type order struct {
	field     string
	direction interface{}
}

type QueryBuilder struct {
	table   string
	joins   []string
	where   interface{}
	args    []interface{}
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

func (qb *QueryBuilder) Table(table string) *QueryBuilder {
	qb.table = table

	return qb
}

func (qb *QueryBuilder) Joins(joins []string) *QueryBuilder {
	qb.joins = funk.UniqString(joins)

	return qb
}

func (qb *QueryBuilder) Where(query interface{}, args ...interface{}) *QueryBuilder {
	qb.where = query
	qb.args = args

	return qb
}

func (qb *QueryBuilder) GroupBy(field ...string) *QueryBuilder {
	qb.groupBy = append(qb.groupBy, field...)

	return qb
}

func (qb *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
	order := order{
		field:     field,
		direction: direction,
	}

	qb.orderBy = append(qb.orderBy, order)

	return qb
}

func (qb *QueryBuilder) Page(offset int, size int) *QueryBuilder {
	qb.page = &page{
		offset: offset,
		limit:  size,
	}

	return qb
}
