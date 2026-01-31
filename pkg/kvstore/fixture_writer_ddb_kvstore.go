package kvstore

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type KvStoreFixture struct {
	Key   any
	Value any
}

func (k KvStoreFixture) GetValue() any {
	return k.Value
}

type dynamoDbKvStoreFixtureWriter[T any] struct {
	logger log.Logger
	store  KvStore[T]
}

func DynamoDbKvStoreFixtureSetFactory[T any](modelId *mdl.ModelId, data fixtures.NamedFixtures[*KvStoreFixture], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewDynamoDbKvStoreFixtureWriter[T](ctx, config, logger, modelId); err != nil {
			return nil, fmt.Errorf("failed to create dynamodb kvstore fixture writer for %s: %w", modelId.String(), err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewDynamoDbKvStoreFixtureWriter[T any](ctx context.Context, config cfg.Config, logger log.Logger, modelId *mdl.ModelId) (fixtures.FixtureWriter, error) {
	if err := modelId.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("failed to pad model id from config: %w", err)
	}

	kvStoreSettings := &Settings{
		ModelId: *modelId,
	}

	var err error
	var store KvStore[T]

	if store, err = NewDdbKvStore[T](ctx, config, logger, kvStoreSettings); err != nil {
		return nil, fmt.Errorf("failed to create dynamodb kv store: %w", err)
	}

	return NewDynamoDbKvStoreFixtureWriterWithInterfaces(logger, store), nil
}

func NewDynamoDbKvStoreFixtureWriterWithInterfaces[T any](logger log.Logger, store KvStore[T]) fixtures.FixtureWriter {
	return &dynamoDbKvStoreFixtureWriter[T]{
		logger: logger,
		store:  store,
	}
}

func (d *dynamoDbKvStoreFixtureWriter[T]) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	m := map[any]any{}

	for _, item := range fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value
	}

	if err := d.store.PutBatch(ctx, m); err != nil {
		return err
	}

	d.logger.Info(ctx, "loaded %d dynamodb kvstore fixtures", len(fixtures))

	return nil
}
