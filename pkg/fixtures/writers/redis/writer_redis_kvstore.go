package redis

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/fixtures/writers"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures/writers/ddb"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type redisKvStoreFixtureWriter struct {
	logger log.Logger
	store  kvstore.KvStore
	purger writers.Purger
}

func RedisKvStoreFixtureWriterFactory(modelId *mdl.ModelId) writers.FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (writers.FixtureWriter, error) {
		settings := &kvstore.Settings{
			AppId: cfg.AppId{
				Project:     modelId.Project,
				Environment: modelId.Environment,
				Family:      modelId.Family,
				Application: modelId.Application,
			},
			Name: modelId.Name,
		}

		store, err := kvstore.NewRedisKvStore(ctx, config, logger, settings)
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

func NewRedisKvStoreFixtureWriterWithInterfaces(logger log.Logger, store kvstore.KvStore, purger writers.Purger) writers.FixtureWriter {
	return &redisKvStoreFixtureWriter{
		logger: logger,
		store:  store,
		purger: purger,
	}
}

func (d *redisKvStoreFixtureWriter) Purge(ctx context.Context) error {
	return d.purger.Purge(ctx)
}

func (d *redisKvStoreFixtureWriter) Write(ctx context.Context, fs *writers.FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	m := map[interface{}]interface{}{}

	for _, item := range fs.Fixtures {
		kvItem := item.(*ddb.KvStoreFixture)
		m[kvItem.Key] = kvItem.Value
	}

	err := d.store.PutBatch(ctx, m)
	if err != nil {
		return err
	}

	d.logger.Info("loaded %d redis kvstore fixtures", len(fs.Fixtures))

	return nil
}
