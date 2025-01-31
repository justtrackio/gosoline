package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	RedisOpRpush = "RPUSH"
	RedisOpSet   = "SET"
)

type RedisFixture struct {
	Key    string
	Value  any
	Expiry time.Duration
}

type redisOpHandler func(ctx context.Context, client Client, fixture *RedisFixture) error

var redisHandlers = map[string]redisOpHandler{
	RedisOpSet: func(ctx context.Context, client Client, fixture *RedisFixture) error {
		return client.Set(ctx, fixture.Key, fixture.Value, fixture.Expiry)
	},
	RedisOpRpush: func(ctx context.Context, client Client, fixture *RedisFixture) error {
		_, err := client.RPush(ctx, fixture.Key, fixture.Value.([]any)...)

		return err
	},
}

type redisFixtureWriter struct {
	logger    log.Logger
	client    Client
	operation string
}

func RedisFixtureSetFactory[T any](name string, operation string, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewRedisFixtureWriter(ctx, config, logger, name, operation); err != nil {
			return nil, fmt.Errorf("failed to create redis fixture writer for %s/%s: %w", name, operation, err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewRedisFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, name string, operation string) (fixtures.FixtureWriter, error) {
	client, err := ProvideClient(ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	return NewRedisFixtureWriterWithInterfaces(logger, client, operation), nil
}

func NewRedisFixtureWriterWithInterfaces(logger log.Logger, client Client, operation string) fixtures.FixtureWriter {
	return &redisFixtureWriter{
		logger:    logger,
		client:    client,
		operation: operation,
	}
}

func (d *redisFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	for _, item := range fixtures {
		redisFixture := item.(*RedisFixture)

		handler, ok := redisHandlers[d.operation]

		if !ok {
			return fmt.Errorf("no handler for operation: %s", d.operation)
		}

		err := handler(ctx, d.client, redisFixture)
		if err != nil {
			return err
		}
	}

	d.logger.Info("loaded %d redis fixtures", len(fixtures))

	return nil
}
