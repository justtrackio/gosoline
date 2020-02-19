package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"reflect"
)

type KvstoreFixture struct {
	Key   interface{}
	Value interface{}
}

type dynamoDbKeyValueFixtureWriter struct {
	config cfg.Config
	logger mon.Logger
}

func NewDynamoDbKvStoreFixtureWriter(cfg cfg.Config, logger mon.Logger) FixtureWriter {
	return cachedWriters.New("dynamoDbKvStore", func() FixtureWriter {
		return &dynamoDbKeyValueFixtureWriter{
			config: cfg,
			logger: logger,
		}
	})
}

func (d *dynamoDbKeyValueFixtureWriter) WriteFixtures(fs *FixtureSet) error {
	modelId, ok := fs.WriterMetadata.(mdl.ModelId)

	if !ok {
		return fmt.Errorf("invalid writer metadata type: %s", reflect.TypeOf(fs.WriterMetadata))
	}

	store := kvstore.NewDdbKvStore(d.config, d.logger, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     modelId.Project,
			Environment: modelId.Environment,
			Family:      modelId.Family,
			Application: modelId.Application,
		},
		Name: modelId.Name,
	})

	for _, item := range fs.Fixtures {
		kvItem, ok := item.(*KvstoreFixture)

		if !ok {
			return fmt.Errorf("invalid fixture type: %s", reflect.TypeOf(item))
		}

		err := store.Put(context.Background(), kvItem.Key, kvItem.Value)

		if err != nil {
			return err
		}
	}

	d.logger.Infof("loaded %d dynamo db kv fixtures", len(fs.Fixtures))

	return nil
}
