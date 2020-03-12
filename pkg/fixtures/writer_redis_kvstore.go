package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
)

type redisKvStoreFixtureWriter struct {
	logger mon.Logger
	store  kvstore.KvStore
	purger *redisPurger
}

func RedisKvStoreFixtureWriterFactory(modelId *mdl.ModelId) FixtureWriterFactory {
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
		store := kvstore.NewRedisKvStore(config, logger, settings)

		name := kvstore.RedisBasename(settings)
		purger := newRedisPurger(config, logger, &name)

		return NewRedisKvStoreFixtureWriterWithInterfaces(logger, store, purger)
	}
}

func NewRedisKvStoreFixtureWriterWithInterfaces(logger mon.Logger, store kvstore.KvStore, purger *redisPurger) FixtureWriter {
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
	for _, item := range fs.Fixtures {
		kvItem := item.(*KvStoreFixture)

		err := d.store.Put(context.Background(), kvItem.Key, kvItem.Value)

		if err != nil {
			return err
		}
	}

	d.logger.Infof("loaded %d redis kvstore fixtures", len(fs.Fixtures))

	return nil
}
