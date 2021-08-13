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
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		kvStoreSettings := &kvstore.Settings{
			AppId: cfg.AppId{
				Project:     modelId.Project,
				Environment: modelId.Environment,
				Family:      modelId.Family,
				Application: modelId.Application,
			},
			Name: modelId.Name,
		}

		kvstoreModel := *modelId
		kvstoreModel.Name = kvstore.DdbBaseName(kvStoreSettings)

		ddbSettings := &ddb.Settings{
			ModelId: kvstoreModel,
		}

		factory := func() (kvstore.KvStore, error) {
			return kvstore.NewDdbKvStore(ctx, config, logger, kvStoreSettings)
		}

		var err error
		var purger *dynamodbPurger

		if purger, err = newDynamodbPurger(ctx, config, logger, ddbSettings); err != nil {
			return nil, fmt.Errorf("can not create dynamodb purger: %w", err)
		}

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

func (d *dynamoDbKvStoreFixtureWriter) Purge(ctx context.Context) error {
	return d.purger.purgeDynamodb(ctx)
}

func (d *dynamoDbKvStoreFixtureWriter) Write(ctx context.Context, fs *FixtureSet) error {
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

	if err = store.PutBatch(ctx, m); err != nil {
		return err
	}

	d.logger.Info("loaded %d dynamodb kvstore fixtures", len(fs.Fixtures))

	return nil
}
