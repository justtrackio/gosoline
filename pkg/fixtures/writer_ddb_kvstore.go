package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
)

type ddbKvstoreFactory func() (kvstore.KvStore, error)

type KvStoreFixture struct {
	Key   interface{}
	Value interface{}
}

type dynamoDbKvStoreFixtureWriter struct {
	logger  log.Logger
	factory ddbKvstoreFactory
	purger  *dynamodbPurger
}

func DynamoDbKvStoreFixtureWriterFactory(modelId *mdl.ModelId) FixtureWriterFactory {
	return func(config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		settings := &kvstore.Settings{
			AppId: cfg.AppId{
				Project:     modelId.Project,
				Environment: modelId.Environment,
				Family:      modelId.Family,
				Application: modelId.Application,
			},
			Name: modelId.Name,
		}

		factory := func() (kvstore.KvStore, error) {
			return kvstore.NewDdbKvStore(config, logger, settings)
		}

		kvstoreModel := *modelId
		kvstoreModel.Name = kvstore.DdbBaseName(settings)

		purger := newDynamodbPurger(config, logger, &ddb.Settings{
			ModelId: kvstoreModel,
		})

		return NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger, factory, purger), nil
	}
}

func NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger log.Logger, factory ddbKvstoreFactory, purger *dynamodbPurger) FixtureWriter {
	return &dynamoDbKvStoreFixtureWriter{
		logger:  logger,
		factory: factory,
		purger:  purger,
	}
}

func (d *dynamoDbKvStoreFixtureWriter) Purge() error {
	return d.purger.purgeDynamodb()
}

func (d *dynamoDbKvStoreFixtureWriter) Write(fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	store, err := d.factory()
	if err != nil {
		return fmt.Errorf("can not create store: %w", err)
	}

	m := map[interface{}]interface{}{}

	for _, item := range fs.Fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value
	}

	if err = store.PutBatch(context.Background(), m); err != nil {
		return err
	}

	d.logger.Info("loaded %d dynamodb kvstore fixtures", len(fs.Fixtures))

	return nil
}
