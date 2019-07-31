package kvstore

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/encoding/msgpack"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"strings"
)

type ddbItem struct {
	Key   string `dynamo:"key,hash"`
	Value []byte `dynamo:"value"`
}

type DdbKvStore struct {
	repository ddb.Repository
	settings   *Settings
}

func NewDdbKvStore(config cfg.Config, logger mon.Logger, settings *Settings) KvStore {
	name := strings.Join([]string{"kvstore", settings.Name}, "-")

	repository := ddb.New(config, logger, ddb.Settings{
		ModelId: mdl.ModelId{
			Name: name,
		},
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
	keyStr := KeyToString(key)

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
	bytes, err := msgpack.Marshal(value)

	if err != nil {
		return err
	}

	keyStr := KeyToString(key)
	item := &ddbItem{
		Key:   keyStr,
		Value: bytes,
	}

	err = s.repository.Save(ctx, item)

	return err
}

func (s *DdbKvStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	keyStr := KeyToString(key)

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

	err = msgpack.Unmarshal(item.Value, value)

	return true, err
}
