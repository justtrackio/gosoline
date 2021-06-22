package kvstore

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/refl"
	"sort"
)

type DdbItem struct {
	Key   string `json:"key" ddb:"key=hash"`
	Value string `json:"value"`
}

type DdbDeleteItem struct {
	Key string `json:"key" ddb:"key=hash"`
}

type ddbKvStore struct {
	repository ddb.Repository
	settings   *Settings
}

func DdbBaseName(settings *Settings) string {
	return fmt.Sprintf("kvstore-%s", settings.Name)
}

func NewDdbKvStore(config cfg.Config, logger log.Logger, settings *Settings) (KvStore, error) {
	settings.PadFromConfig(config)
	name := DdbBaseName(settings)

	repository, err := ddb.NewRepository(config, logger, &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     settings.Project,
			Environment: settings.Environment,
			Family:      settings.Family,
			Application: settings.Application,
			Name:        name,
		},
		Main: ddb.MainSettings{
			Model:              DdbItem{},
			ReadCapacityUnits:  5,
			WriteCapacityUnits: 5,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("can not create ddb repository: %w", err)
	}

	return NewDdbKvStoreWithInterfaces(repository, settings), nil
}

func NewDdbKvStoreWithInterfaces(repository ddb.Repository, settings *Settings) KvStore {
	return NewMetricStoreWithInterfaces(&ddbKvStore{
		repository: repository,
		settings:   settings,
	}, settings)
}

func (s *ddbKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return false, fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
	}

	item := &DdbItem{}
	qb := s.repository.GetItemBuilder().WithHash(keyStr)
	res, err := s.repository.GetItem(ctx, qb, item)

	if err != nil {
		return false, fmt.Errorf("can not check if ddb store contains the key %s: %w", keyStr, err)
	}

	return res.IsFound, nil
}

func (s *ddbKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return false, fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
	}

	qb := s.repository.GetItemBuilder().WithHash(keyStr)

	item := &DdbItem{}
	res, err := s.repository.GetItem(ctx, qb, item)

	if err != nil {
		return false, fmt.Errorf("can not get item %s from ddb store: %w", keyStr, err)
	}

	if !res.IsFound {
		return false, nil
	}

	bytes := []byte(item.Value)
	err = Unmarshal(bytes, value)

	if err != nil {
		return false, fmt.Errorf("can not unmarshal value for item %s: %w", keyStr, err)
	}

	return true, nil
}

func (s *ddbKvStore) GetBatch(ctx context.Context, keys interface{}, result interface{}) ([]interface{}, error) {
	return getBatch(ctx, keys, result, s.getChunk, s.settings.BatchSize)
}

func (s *ddbKvStore) getChunk(ctx context.Context, resultMap *refl.Map, keys []interface{}) ([]interface{}, error) {
	var err error

	keyStrings := make([]string, len(keys))
	keyMapToOriginal := make(map[string]interface{}, len(keys))

	for i := 0; i < len(keyStrings); i++ {
		keyStr, err := CastKeyToString(keys[i])

		if err != nil {
			return nil, fmt.Errorf("can not cast key %T %v to string: %w", keys[i], keys[i], err)
		}

		keyStrings[i] = keyStr
		keyMapToOriginal[keyStr] = keys[i]
	}

	qb := s.repository.BatchGetItemsBuilder()
	qb.WithHashKeys(keyStrings)
	items := make([]DdbItem, 0)

	_, err = s.repository.BatchGetItems(ctx, qb, &items)

	if err != nil {
		return nil, fmt.Errorf("can not get items from ddb: %w", err)
	}

	found := make(map[string]bool)

	for i := 0; i < len(items); i++ {
		found[items[i].Key] = true

		element := resultMap.NewElement()
		err = Unmarshal([]byte(items[i].Value), element)

		if err != nil {
			return nil, fmt.Errorf("can not unmarshal item: %w", err)
		}

		keyOrig := keyMapToOriginal[items[i].Key]
		if err := resultMap.Set(keyOrig, element); err != nil {
			return nil, fmt.Errorf("can not set new element on result map: %w", err)
		}
	}

	missing := make([]interface{}, 0)

	for i, key := range keyStrings {
		if !found[key] {
			missing = append(missing, keys[i])
		}
	}

	return missing, nil
}

func (s *ddbKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
	}

	bytes, err := Marshal(value)

	if err != nil {
		return fmt.Errorf("can not marshal value %s: %w", keyStr, err)
	}

	item := &DdbItem{
		Key:   keyStr,
		Value: string(bytes),
	}

	_, err = s.repository.PutItem(ctx, nil, item)

	if err != nil {
		return fmt.Errorf("can not put item %s into ddb store: %w", keyStr, err)
	}

	return nil
}

func (s *ddbKvStore) PutBatch(ctx context.Context, values interface{}) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return fmt.Errorf("could not convert values to map[interface{}]interface{}")
	}

	keyStrings := make([]string, 0, len(mii))
	keyMap := make(map[string]interface{})

	for k := range mii {
		keyStr, err := CastKeyToString(k)

		if err != nil {
			return fmt.Errorf("can not cast key %T %v to string: %w", k, k, err)
		}

		keyStrings = append(keyStrings, keyStr)
		keyMap[keyStr] = k
	}

	sort.Strings(keyStrings)
	items := make([]DdbItem, 0, len(mii))

	for _, keyStr := range keyStrings {
		key := keyMap[keyStr]
		value := mii[key]

		bytes, err := Marshal(value)

		if err != nil {
			return fmt.Errorf("can not marshal value %s: %w", keyStr, err)
		}

		item := DdbItem{
			Key:   keyStr,
			Value: string(bytes),
		}

		items = append(items, item)
	}

	_, err = s.repository.BatchPutItems(ctx, items)

	if err != nil {
		return fmt.Errorf("not able to put values into ddb store: %w", err)
	}

	return nil
}

func (s *ddbKvStore) Delete(ctx context.Context, key interface{}) error {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		return fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
	}

	_, err = s.repository.DeleteItem(ctx, nil, &DdbDeleteItem{
		Key: keyStr,
	})

	if err != nil {
		return fmt.Errorf("can not delete item %s from ddb store: %w", keyStr, err)
	}

	return nil
}

func (s *ddbKvStore) DeleteBatch(ctx context.Context, keys interface{}) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []interface{}: %w", keys, err)
	}

	items := make([]*DdbDeleteItem, len(si))

	for i, key := range si {
		keyStr, err := CastKeyToString(key)

		if err != nil {
			return fmt.Errorf("can not cast key %T %v to string: %w", key, key, err)
		}

		items[i] = &DdbDeleteItem{
			Key: keyStr,
		}
	}

	_, err = s.repository.BatchDeleteItems(ctx, items)

	if err != nil {
		return fmt.Errorf("can not delete values from ddb store: %w", err)
	}

	return nil
}
