package ddb

//go:generate mockery -name QueryBuilderSimple
type QueryBuilderSimple interface {
	WithHash(value interface{}) QueryBuilderSimple
	WithRange(comp string, values ...interface{}) QueryBuilderSimple
	Build() QueryBuilder
}

type queryBuilderSimple struct {
	base QueryBuilder
}

func NewQueryBuilderSimple(base QueryBuilder) QueryBuilderSimple {
	return &queryBuilderSimple{
		base: base,
	}
}

func (b *queryBuilderSimple) WithHash(value interface{}) QueryBuilderSimple {
	b.base.WithHash(value)

	return b
}

func (b *queryBuilderSimple) WithRange(comp string, values ...interface{}) QueryBuilderSimple {
	b.base.WithRange(comp, values...)

	return b
}

func (b *queryBuilderSimple) Build() QueryBuilder {
	return b.base
}
