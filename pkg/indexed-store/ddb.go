package indexed_store

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"reflect"
)

type ddbStore struct {
	repo       ddb.Repository
	modelType  reflect.Type
	indexTypes map[string]reflect.Type
}

func NewDdbIndexedStore(config cfg.Config, logger log.Logger, settings *Settings) (IndexedStore, error) {
	settings.PadFromConfig(config)

	globalSettings := make([]ddb.GlobalSettings, len(settings.Indices))
	for i, index := range settings.Indices {
		globalSettings[i] = ddb.GlobalSettings{
			Name:               index.Name,
			Model:              index.Model,
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		}
	}

	repo, err := ddb.NewRepository(config, logger, &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     settings.Project,
			Environment: settings.Environment,
			Family:      settings.Family,
			Application: settings.Application,
			Name:        fmt.Sprintf("indexed-store-%s", settings.Name),
		},
		Main: ddb.MainSettings{
			Model:              settings.Model,
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
		Global: globalSettings,
	})
	if err != nil {
		return nil, fmt.Errorf("can not create ddb repo: %w", err)
	}

	return NewDdbIndexedStoreFromInterfaces(repo, settings), nil
}

func NewDdbIndexedStoreFromInterfaces(repo ddb.Repository, settings *Settings) IndexedStore {
	indexTypes := map[string]reflect.Type{}

	for _, index := range settings.Indices {
		indexTypes[index.Name] = reflect.SliceOf(reflect.TypeOf(index.Model))
	}

	return &ddbStore{
		repo:       repo,
		modelType:  reflect.TypeOf(settings.Model),
		indexTypes: indexTypes,
	}
}

func (s *ddbStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	qb := s.repo.GetItemBuilder().WithHash(key)
	result, err := s.repo.GetItem(ctx, qb, s.getModel())

	if err != nil {
		return false, fmt.Errorf("failed to search for item: %w", err)
	}

	return result.IsFound, nil
}

func (s *ddbStore) ContainsInIndex(ctx context.Context, index string, key interface{}, rangeKeys ...interface{}) (bool, error) {
	qb := s.repo.QueryBuilder().WithIndex(index).WithHash(key).WithLimit(1)
	switch len(rangeKeys) {
	case 0:
		break
	case 1:
		qb = qb.WithRangeEq(rangeKeys[0])
	default:
		return false, fmt.Errorf("an indexed ddb store can only query with up to a single range key, got %d keys", len(rangeKeys))
	}

	modelSlice, err := s.getIndexModel(index)
	if err != nil {
		return false, err
	}

	result, err := s.repo.Query(ctx, qb, &modelSlice)

	if err != nil {
		return false, fmt.Errorf("failed to search for item in index %s: %w", index, err)
	}

	return result.ItemCount > 0, nil
}

func (s *ddbStore) Get(ctx context.Context, key interface{}) (BaseValue, error) {
	qb := s.repo.GetItemBuilder().WithHash(key)
	modelPtr := s.getModel()
	result, err := s.repo.GetItem(ctx, qb, modelPtr)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve item: %w", err)
	}

	if !result.IsFound {
		return nil, nil
	}

	model := reflect.ValueOf(modelPtr).Elem().Interface().(BaseValue)

	return model, nil
}

func (s *ddbStore) GetFromIndex(ctx context.Context, index string, key interface{}, rangeKeys ...interface{}) (BaseValue, error) {
	qb := s.repo.QueryBuilder().WithIndex(index).WithHash(key).WithLimit(1)
	switch len(rangeKeys) {
	case 0:
		break
	case 1:
		qb = qb.WithRangeEq(rangeKeys[0])
	default:
		return nil, fmt.Errorf("an indexed ddb store can only query with up to a single range key, got %d keys", len(rangeKeys))
	}

	modelSlice, err := s.getIndexModel(index)
	if err != nil {
		return nil, err
	}

	result, err := s.repo.Query(ctx, qb, modelSlice)

	if err != nil {
		return nil, fmt.Errorf("failed to search for item in index %s: %w", index, err)
	}

	if result.ItemCount == 0 {
		return nil, nil
	}

	model := reflect.ValueOf(modelSlice).Elem().Index(0).Interface().(IndexValue)

	return model.ToBaseValue(), nil
}

func (s *ddbStore) GetBatch(ctx context.Context, keys interface{}) ([]BaseValue, error) {
	panic("implement me")
}

func (s *ddbStore) GetBatchWithMissing(ctx context.Context, keys interface{}) ([]BaseValue, []interface{}, error) {
	panic("implement me")
}

func (s *ddbStore) GetBatchFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, error) {
	panic("implement me")
}

func (s *ddbStore) GetBatchWithMissingFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, []MissingValue, error) {
	panic("implement me")
}

func (s *ddbStore) Put(ctx context.Context, value BaseValue) error {
	_, err := s.repo.PutItem(ctx, s.repo.PutItemBuilder(), value)

	if err != nil {
		return fmt.Errorf("failed to write item to ddb: %w", err)
	}

	return nil
}

func (s *ddbStore) PutBatch(ctx context.Context, values interface{}) error {
	panic("implement me")
}

func (s *ddbStore) Delete(ctx context.Context, key interface{}) error {
	_, err := s.repo.DeleteItem(ctx, s.repo.DeleteItemBuilder().WithHash(key), s.getModel())

	if err != nil {
		return fmt.Errorf("failed to delete item from ddb: %w", err)
	}

	return nil
}

func (s *ddbStore) DeleteBatch(ctx context.Context, keys interface{}) error {
	panic("implement me")
}

func (s *ddbStore) getModel() interface{} {
	return reflect.New(s.modelType).Interface()
}

func (s *ddbStore) getIndexModel(index string) (interface{}, error) {
	if typ, ok := s.indexTypes[index]; !ok {
		return nil, fmt.Errorf("unknown index %s", index)
	} else {
		return reflect.New(typ).Interface(), nil
	}
}
