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
	config  cfg.Config
	logger  mon.Logger
	modelId *mdl.ModelId
}

func DynamoDbKvStoreFixtureWriterFactory(modelId *mdl.ModelId) FixtureWriterFactory {
	return func(cfg cfg.Config, logger mon.Logger) FixtureWriter {
		writer := &dynamoDbKeyValueFixtureWriter{
			config:  cfg,
			logger:  logger,
			modelId: modelId,
		}

		return writer
	}
}

func (d *dynamoDbKeyValueFixtureWriter) WriteFixtures(fs *FixtureSet) error {
	store := kvstore.NewDdbKvStore(d.config, d.logger, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     d.modelId.Project,
			Environment: d.modelId.Environment,
			Family:      d.modelId.Family,
			Application: d.modelId.Application,
		},
		Name: d.modelId.Name,
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
