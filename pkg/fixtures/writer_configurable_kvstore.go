package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
)

type configurableKvStoreFixtureWriter[T any] struct {
	logger log.Logger
	store  kvstore.KvStore[T]
}

func ConfigurableKvStoreFixtureWriterFactory[T any](name string) FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		store, err := kvstore.ProvideConfigurableKvStore[T](ctx, config, logger, name)
		if err != nil {
			return nil, fmt.Errorf("can not provide configurable kvstore: %w", err)
		}

		return NewConfigurableKvStoreFixtureWriterWithInterfaces[T](logger, store), nil
	}
}

func NewConfigurableKvStoreFixtureWriterWithInterfaces[T any](logger log.Logger, store kvstore.KvStore[T]) FixtureWriter {
	return &configurableKvStoreFixtureWriter[T]{
		logger: logger,
		store:  store,
	}
}

func (c *configurableKvStoreFixtureWriter[T]) Purge(_ context.Context) error {
	c.logger.Info("purging configurable kvstore not supported")
	return nil
}

func (c *configurableKvStoreFixtureWriter[T]) Write(ctx context.Context, fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	m := map[interface{}]T{}

	for _, item := range fs.Fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value.(T)
	}

	err := c.store.PutBatch(ctx, m)
	if err != nil {
		return err
	}

	c.logger.Info("loaded %d configurable kvstore fixtures", len(fs.Fixtures))

	return nil
}
