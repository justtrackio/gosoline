package ddb

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/fixtures/writers"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type KvStoreFixture struct {
	Key   interface{}
	Value interface{}
}

type ddbKvstoreFactory func() (kvstore.KvStore, error)

type dynamoDbKvStoreFixtureWriter struct {
	logger  log.Logger
	factory ddbKvstoreFactory
	purger  writers.Purger
}

func DynamoDbKvStoreFixtureWriterFactory(modelId *mdl.ModelId) writers.FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (writers.FixtureWriter, error) {
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
		var purger writers.Purger

		if purger, err = newDynamodbPurger(ctx, config, logger, ddbSettings); err != nil {
			return nil, fmt.Errorf("can not create dynamodb purger: %w", err)
		}

		return NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger, factory, purger), nil
	}
}

func NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger log.Logger, factory ddbKvstoreFactory, purger writers.Purger) writers.FixtureWriter {
	return &dynamoDbKvStoreFixtureWriter{
		logger:  logger,
		factory: factory,
		purger:  purger,
	}
}

func (d *dynamoDbKvStoreFixtureWriter) Purge(ctx context.Context) error {
	return d.purger.Purge(ctx)
}

func (d *dynamoDbKvStoreFixtureWriter) Write(ctx context.Context, fs *writers.FixtureSet) error {
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
