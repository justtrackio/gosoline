package kvstore

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"strings"
)

type ddbItem struct {
	Key   string `dynamo:"key,hash"`
	Value string `dynamo:"value"`
}

type DdbKvStore struct {
	repository ddb.Repository
	settings   *Settings
}

func NewDdbKvStore(config cfg.Config, logger mon.Logger, settings *Settings) KvStore {
	settings.PadFromConfig(config)

	name := strings.Join([]string{"kvstore", settings.Name}, "-")
	modelId := mdl.ModelId{
		Project:     settings.Project,
		Environment: settings.Environment,
		Family:      settings.Family,
		Application: settings.Application,
		Name:        name,
	}

	repository := ddb.New(config, logger, ddb.Settings{
		ModelId:            modelId,
		ReadCapacityUnits:  5,
		WriteCapacityUnits: 5,
	})

	err := repository.CreateTable(ddbItem{})

	if err != nil {
		panic(err)
	}

	return NewDdbKvStoreWithInterfaces(repository, settings)
}

func NewDdbKvStoreWithInterfaces(repository ddb.Repository, settings *Settings) *DdbKvStore {
	return &DdbKvStore{
		repository: repository,
		settings:   settings,
	}
}

func (s *DdbKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	keyStr, err := KeyToString(key)

	if err != nil {
		return false, err
	}

	qb := s.repository.QueryBuilder()
	qb.WithHash("key", keyStr)

	item := &ddbItem{}
	exists, err := s.repository.GetItem(ctx, qb, &item)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func (s *DdbKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	bytes, err := Marshal(value)

	if err != nil {
		return err
	}

	keyStr, err := KeyToString(key)

	if err != nil {
		return err
	}

	item := &ddbItem{
		Key:   keyStr,
		Value: string(bytes),
	}

	err = s.repository.Save(ctx, item)

	return err
}

func (s *DdbKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr, err := KeyToString(key)

	if err != nil {
		return false, err
	}

	qb := s.repository.QueryBuilder()
	qb.WithHash("key", keyStr)

	item := &ddbItem{}
	exists, err := s.repository.GetItem(ctx, qb, &item)

	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	bytes := []byte(item.Value)
	err = Unmarshal(bytes, value)

	return true, err
}
