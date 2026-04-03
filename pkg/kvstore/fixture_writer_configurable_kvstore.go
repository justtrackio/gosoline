package kvstore

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

type configurableKvStoreFixtureWriter[T any] struct {
	logger log.Logger
	store  KvStore[T]
}

func ConfigurableKvStoreFixtureSetFactory[T any](name string, data fixtures.NamedFixtures[*KvStoreFixture], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewConfigurableKvStoreFixtureWriter[T](ctx, config, logger, name); err != nil {
			return nil, fmt.Errorf("failed to create configurable kvstore fixture writer for %s: %w", name, err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func configurableKvStoreResourceIds(config cfg.Config, name string) ([]string, error) {
	key := fmt.Sprintf("kvstore.%s.type", name)
	t, err := config.GetString(key)
	if err != nil {
		return nil, fmt.Errorf("could not get type for kvstore %s: %w", name, err)
	}

	if t != TypeChain {
		return nil, fmt.Errorf("invalid kvstore %s of type %s, expected type %s", name, t, TypeChain)
	}

	configuration := ChainConfiguration{}
	if err := config.UnmarshalKey(GetConfigurableKey(name), &configuration); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kvstore configuration for %s: %w", name, err)
	}

	resourceIds := make([]string, 0, len(configuration.Elements))
	for _, element := range configuration.Elements {
		switch element {
		case TypeDdb:
			modelId := configuration.ModelId
			modelId.Name = DdbBaseName(&Settings{ModelId: modelId, DdbSettings: configuration.Ddb})
			resourceIds = append(resourceIds, fmt.Sprintf("ddb/%s", modelId.String()))
		case TypeRedis:
			resourceIds = append(resourceIds, fmt.Sprintf("redis/%s", RedisBasename(name)))
		}
	}

	return resourceIds, nil
}

func NewConfigurableKvStoreFixtureWriter[T any](ctx context.Context, config cfg.Config, logger log.Logger, name string) (fixtures.FixtureWriter, error) {
	store, err := ProvideConfigurableKvStore[T](ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not provide configurable kvstore: %w", err)
	}

	resourceIds, err := configurableKvStoreResourceIds(config, name)
	if err != nil {
		return nil, fmt.Errorf("failed to determine configurable kvstore resources for %s: %w", name, err)
	}

	return fixtures.NewManagedFixtureWriter(NewConfigurableKvStoreFixtureWriterWithInterfaces[T](logger, store), resourceIds...), nil
}

func NewConfigurableKvStoreFixtureWriterWithInterfaces[T any](logger log.Logger, store KvStore[T]) fixtures.FixtureWriter {
	return &configurableKvStoreFixtureWriter[T]{
		logger: logger,
		store:  store,
	}
}

func (c *configurableKvStoreFixtureWriter[T]) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	m := map[any]T{}

	for _, item := range fixtures {
		kvItem := item.(*KvStoreFixture)
		m[kvItem.Key] = kvItem.Value.(T)
	}

	err := c.store.PutBatch(ctx, m)
	if err != nil {
		return err
	}

	c.logger.Info(ctx, "loaded %d configurable kvstore fixtures", len(fixtures))

	return nil
}
