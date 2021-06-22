package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
)

type redisKvStoreFixtureWriter struct {
	logger log.Logger
	store  kvstore.KvStore
	purger *redisPurger
}

func RedisKvStoreFixtureWriterFactory(modelId *mdl.ModelId) FixtureWriterFactory {
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

		store, err := kvstore.NewRedisKvStore(config, logger, settings)
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

func NewRedisKvStoreFixtureWriterWithInterfaces(logger log.Logger, store kvstore.KvStore, purger *redisPurger) FixtureWriter {
	return &redisKvStoreFixtureWriter{
		logger: logger,
		store:  store,
		purger: purger,
	}
}

func (d *redisKvStoreFixtureWriter) Purge() error {
	return d.purger.purge()
}

func (d *redisKvStoreFixtureWriter) Write(fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	m := map[interface{}]interface{}{}

	for _, item := range fs.Fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value
	}

	err := d.store.PutBatch(context.Background(), m)
	if err != nil {
		return err
	}

	d.logger.Info("loaded %d redis kvstore fixtures", len(fs.Fixtures))

	return nil
}
