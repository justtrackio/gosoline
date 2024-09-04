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

func ConfigurableKvStoreFixtureSetFactory[T any, T2 any](name string, data NamedFixtures[T], options ...FixtureSetOption) FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureSet, error) {
		var err error
		var writer FixtureWriter

		if writer, err = NewConfigurableKvStoreFixtureWriter[T2](ctx, config, logger, name); err != nil {
			return nil, fmt.Errorf("failed to create configurable kvstore fixture writer for %s: %w", name, err)
		}

		return NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewConfigurableKvStoreFixtureWriter[T any](ctx context.Context, config cfg.Config, logger log.Logger, name string) (FixtureWriter, error) {
	store, err := kvstore.ProvideConfigurableKvStore[T](ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not provide configurable kvstore: %w", err)
	}

	return NewConfigurableKvStoreFixtureWriterWithInterfaces[T](logger, store), nil
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

func (c *configurableKvStoreFixtureWriter[T]) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	m := map[interface{}]T{}

	for _, item := range fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value.(T)
	}

	err := c.store.PutBatch(ctx, m)
	if err != nil {
		return err
	}

	c.logger.Info("loaded %d configurable kvstore fixtures", len(fixtures))

	return nil
}
