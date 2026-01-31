package kvstore

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type redisKvStoreFixtureWriter[T any] struct {
	logger log.Logger
	store  KvStore[T]
}

func RedisKvStoreFixtureSetFactory[T any](modelId *mdl.ModelId, data fixtures.NamedFixtures[*KvStoreFixture], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewRedisKvStoreFixtureWriter[T](ctx, config, logger, modelId); err != nil {
			return nil, fmt.Errorf("failed to create redis kvstore fixture writer for %s: %w", modelId.String(), err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewRedisKvStoreFixtureWriter[T any](ctx context.Context, config cfg.Config, logger log.Logger, modelId *mdl.ModelId) (fixtures.FixtureWriter, error) {
	settings := &Settings{
		ModelId: *modelId,
	}

	store, err := NewRedisKvStore[T](ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create redis store: %w", err)
	}

	return NewRedisKvStoreFixtureWriterWithInterfaces(logger, store), nil
}

func NewRedisKvStoreFixtureWriterWithInterfaces[T any](logger log.Logger, store KvStore[T]) fixtures.FixtureWriter {
	return &redisKvStoreFixtureWriter[T]{
		logger: logger,
		store:  store,
	}
}

func (d *redisKvStoreFixtureWriter[T]) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	m := map[any]any{}

	for _, item := range fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value
	}

	err := d.store.PutBatch(ctx, m)
	if err != nil {
		return err
	}

	d.logger.Info(ctx, "loaded %d redis kvstore fixtures", len(fixtures))

	return nil
}
