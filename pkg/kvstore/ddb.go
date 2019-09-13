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
	Key   string `json:"key" ddb:"key=hash"`
	Value string `json:"value"`
}

type DdbKvStore struct {
	logger     mon.Logger
	repository ddb.Repository
	settings   *Settings
}

func NewDdbKvStore(config cfg.Config, logger mon.Logger, settings *Settings) KvStore {
	settings.PadFromConfig(config)
	name := strings.Join([]string{"kvstore", settings.Name}, "-")

	repository := ddb.NewRepository(config, logger, &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     settings.Project,
			Environment: settings.Environment,
			Family:      settings.Family,
			Application: settings.Application,
			Name:        name,
		},
		Main: ddb.MainSettings{
			Model:              ddbItem{},
			ReadCapacityUnits:  5,
			WriteCapacityUnits: 5,
		},
	})

	return NewDdbKvStoreWithInterfaces(logger, repository, settings)
}

func NewDdbKvStoreWithInterfaces(logger mon.Logger, repository ddb.Repository, settings *Settings) *DdbKvStore {
	return &DdbKvStore{
		logger:     logger,
		repository: repository,
		settings:   settings,
	}
}

func (s *DdbKvStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		s.logger.Error(err, "can not cast key to string")
		return false, err
	}

	item := &ddbItem{}
	qb := s.repository.GetItemBuilder().WithHash(keyStr)
	res, err := s.repository.GetItem(ctx, qb, &item)

	if err != nil {
		s.logger.Error(err, "can not check if ddb store contains the key")
		return false, err
	}

	return res.IsFound, nil
}

func (s *DdbKvStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	bytes, err := Marshal(value)

	if err != nil {
		s.logger.Error(err, "can not marshal value")
		return err
	}

	keyStr, err := CastKeyToString(key)

	if err != nil {
		s.logger.Error(err, "can not cast key to string")
		return err
	}

	item := &ddbItem{
		Key:   keyStr,
		Value: string(bytes),
	}

	_, err = s.repository.PutItem(ctx, nil, item)

	if err != nil {
		s.logger.Error(err, "can not put value into ddb store")
		return err
	}

	return nil
}

func (s *DdbKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr, err := CastKeyToString(key)

	if err != nil {
		s.logger.Error(err, "can not cast key to string")
		return false, err
	}

	qb := s.repository.GetItemBuilder().WithHash(keyStr)

	item := &ddbItem{}
	res, err := s.repository.GetItem(ctx, qb, &item)

	if err != nil {
		s.logger.Error(err, "can not get item from ddb store")
		return false, err
	}

	if !res.IsFound {
		return false, nil
	}

	bytes := []byte(item.Value)
	err = Unmarshal(bytes, value)

	if err != nil {
		s.logger.Error(err, "can not unmarshal value")
		return false, err
	}

	return true, nil
}
