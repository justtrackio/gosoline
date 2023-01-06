package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type redisKvStoreFixtureWriter[T any] struct {
	logger log.Logger
	store  kvstore.KvStore[T]
	purger *redisPurger
}

func RedisKvStoreFixtureWriterFactory[T any](modelId *mdl.ModelId) FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		settings := &kvstore.Settings{
			AppId: cfg.AppId{
				Project:     modelId.Project,
				Environment: modelId.Environment,
				Family:      modelId.Family,
				Group:       modelId.Group,
				Application: modelId.Application,
			},
			Name: modelId.Name,
		}

		store, err := kvstore.NewRedisKvStore[T](ctx, config, logger, settings)
		if err != nil {
			return nil, fmt.Errorf("can not create redis store: %w", err)
		}

		name := kvstore.RedisBasename(settings)

		purger, err := newRedisPurger(config, logger, &name)
		if err != nil {
			return nil, fmt.Errorf("can not create redis purger: %w", err)
		}

		return NewRedisKvStoreFixtureWriterWithInterfaces(logger, store, purger), nil
	}
}

func NewRedisKvStoreFixtureWriterWithInterfaces[T any](logger log.Logger, store kvstore.KvStore[T], purger *redisPurger) FixtureWriter {
	return &redisKvStoreFixtureWriter[T]{
		logger: logger,
		store:  store,
		purger: purger,
	}
}

func (d *redisKvStoreFixtureWriter[T]) Purge(ctx context.Context) error {
	return d.purger.purge(ctx)
}

func (d *redisKvStoreFixtureWriter[T]) Write(ctx context.Context, fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	m := map[interface{}]interface{}{}

	for _, item := range fs.Fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value
	}

	err := d.store.PutBatch(ctx, m)
	if err != nil {
		return err
	}

	d.logger.Info("loaded %d redis kvstore fixtures", len(fs.Fixtures))

	return nil
}
