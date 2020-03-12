package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
)

type ddbKvstoreFactory func() kvstore.KvStore

type KvStoreFixture struct {
	Key   interface{}
	Value interface{}
}

type dynamoDbKvStoreFixtureWriter struct {
	logger  mon.Logger
	factory ddbKvstoreFactory
	purger  *dynamodbPurger
}

func DynamoDbKvStoreFixtureWriterFactory(modelId *mdl.ModelId) FixtureWriterFactory {
	return func(config cfg.Config, logger mon.Logger) FixtureWriter {
		settings := &kvstore.Settings{
			AppId: cfg.AppId{
				Project:     modelId.Project,
				Environment: modelId.Environment,
				Family:      modelId.Family,
				Application: modelId.Application,
			},
			Name: modelId.Name,
		}
		factory := func() kvstore.KvStore {
			return kvstore.NewDdbKvStore(config, logger, settings)
		}

		kvstoreModel := *modelId
		kvstoreModel.Name = kvstore.DdbBaseName(settings)

		purger := newDynamodbPurger(config, logger, &ddb.Settings{
			ModelId: kvstoreModel,
		})

		return NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger, factory, purger)
	}
}

func NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger mon.Logger, factory ddbKvstoreFactory, purger *dynamodbPurger) FixtureWriter {
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
	store := d.factory()

	for _, item := range fs.Fixtures {
		kvItem := item.(*KvStoreFixture)

		err := store.Put(context.Background(), kvItem.Key, kvItem.Value)

		if err != nil {
			return err
		}
	}

	d.logger.Infof("loaded %d dynamodb kvstore fixtures", len(fs.Fixtures))

	return nil
}
