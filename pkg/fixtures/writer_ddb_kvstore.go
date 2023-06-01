package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type ddbKvstoreFactory[T any] func() (kvstore.KvStore[T], error)

type KvStoreFixture struct {
	Key   interface{}
	Value interface{}
}

type dynamoDbKvStoreFixtureWriter[T any] struct {
	logger  log.Logger
	factory ddbKvstoreFactory[T]
	purger  *dynamodbPurger
}

func DynamoDbKvStoreFixtureWriterFactory[T any](modelId *mdl.ModelId) FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		kvStoreSettings := &kvstore.Settings{
			AppId: cfg.AppId{
				Project:     modelId.Project,
				Environment: modelId.Environment,
				Family:      modelId.Family,
				Group:       modelId.Group,
				Application: modelId.Application,
			},
			Name: modelId.Name,
		}

		kvstoreModel := *modelId
		kvstoreModel.Name = kvstore.DdbBaseName(kvStoreSettings)

		ddbSettings := &ddb.Settings{
			ModelId: kvstoreModel,
		}

		factory := func() (kvstore.KvStore[T], error) {
			return kvstore.NewDdbKvStore[T](ctx, config, logger, kvStoreSettings)
		}

		var err error
		var purger *dynamodbPurger

		if purger, err = newDynamodbPurger(ctx, config, logger, ddbSettings); err != nil {
			return nil, fmt.Errorf("can not create dynamodb purger: %w", err)
		}

		return NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger, factory, purger), nil
	}
}

func NewDynamoDbKvStoreFixtureWriterWithInterfaces[T any](logger log.Logger, factory ddbKvstoreFactory[T], purger *dynamodbPurger) FixtureWriter {
	return &dynamoDbKvStoreFixtureWriter[T]{
		logger:  logger,
		factory: factory,
		purger:  purger,
	}
}

func (d *dynamoDbKvStoreFixtureWriter[T]) Purge(ctx context.Context) error {
	return d.purger.purgeDynamodb(ctx)
}

func (d *dynamoDbKvStoreFixtureWriter[T]) Write(ctx context.Context, fs *FixtureSet) error {
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
