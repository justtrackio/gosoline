package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/log"
)

type configurableKvStoreFixtureWriter struct {
	logger log.Logger
	store  kvstore.KvStore
}

func ConfigurableKvStoreFixtureWriterFactory(name string) FixtureWriterFactory {
	return func(config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		store, err := kvstore.ProvideConfigurableKvStore(config, logger, name)
		if err != nil {
			return nil, fmt.Errorf("can not provide configurable kvstore: %w", err)
		}

		return NewConfigurableKvStoreFixtureWriterWithInterfaces(logger, store), nil
	}
}

func NewConfigurableKvStoreFixtureWriterWithInterfaces(logger log.Logger, store kvstore.KvStore) FixtureWriter {
	return &configurableKvStoreFixtureWriter{
		logger: logger,
		store:  store,
	}
}

func (c *configurableKvStoreFixtureWriter) Purge() error {
	c.logger.Info("purging configurable kvstore not supported")
	return nil
}

func (c *configurableKvStoreFixtureWriter) Write(fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	m := map[interface{}]interface{}{}

	for _, item := range fs.Fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value
	}

	err := c.store.PutBatch(context.Background(), m)
	if err != nil {
		return err
	}

	c.logger.Info("loaded %d configurable kvstore fixtures", len(fs.Fixtures))

	return nil
}
