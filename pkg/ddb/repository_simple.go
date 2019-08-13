package ddb

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type SimpleRepository interface {
	DeleteItem(ctx context.Context, result interface{}) (*DeleteItemResult, error)
	GetItem(ctx context.Context, result interface{}) (*GetItemResult, error)
	PutItem(ctx context.Context, result interface{}) (*PutItemResult, error)
	Query(ctx context.Context, qb QueryBuilderSimple, result interface{}) (*QueryResult, error)
	QueryBuilder() QueryBuilderSimple
}

type simpleRepository struct {
	base Repository
}

func NewSimpleRepository(config cfg.Config, logger mon.Logger, settings *SimpleSettings) *simpleRepository {
	baseSettings := &Settings{
		ModelId: settings.ModelId,
		Main: MainSettings{
			Model:              settings.Model,
			StreamView:         settings.StreamView,
			ReadCapacityUnits:  settings.ReadCapacityUnits,
			WriteCapacityUnits: settings.WriteCapacityUnits,
		},
	}

	base := NewRepository(config, logger, baseSettings)

	return NewSimpleRepositoryWithInterfaces(base)
}

func NewSimpleRepositoryWithInterfaces(base Repository) *simpleRepository {
	return &simpleRepository{
		base: base,
	}
}

func (r *simpleRepository) DeleteItem(ctx context.Context, result interface{}) (*DeleteItemResult, error) {
	return r.base.DeleteItem(ctx, nil, result)
}

func (r *simpleRepository) GetItem(ctx context.Context, result interface{}) (*GetItemResult, error) {
	return r.base.GetItem(ctx, nil, result)
}

func (r *simpleRepository) PutItem(ctx context.Context, result interface{}) (*PutItemResult, error) {
	return r.base.PutItem(ctx, nil, result)
}

func (r *simpleRepository) Query(ctx context.Context, qb QueryBuilderSimple, result interface{}) (*QueryResult, error) {
	base := qb.Build()

	return r.base.Query(ctx, base, result)
}

func (r *simpleRepository) QueryBuilder() QueryBuilderSimple {
	base := r.base.QueryBuilder()
	qb := NewQueryBuilderSimple(base)

	return qb
}
