package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/redis"
)

const (
	RedisOpRpush = "RPUSH"
	RedisOpSet   = "SET"
)

type RedisFixture struct {
	Key    string
	Value  interface{}
	Expiry time.Duration
}

type redisOpHandler func(ctx context.Context, client redis.Client, fixture *RedisFixture) error

var redisHandlers = map[string]redisOpHandler{
	RedisOpSet: func(ctx context.Context, client redis.Client, fixture *RedisFixture) error {
		return client.Set(ctx, fixture.Key, fixture.Value, fixture.Expiry)
	},
	RedisOpRpush: func(ctx context.Context, client redis.Client, fixture *RedisFixture) error {
		_, err := client.RPush(ctx, fixture.Key, fixture.Value.([]interface{})...)

		return err
	},
}

type redisFixtureWriter struct {
	logger    log.Logger
	client    redis.Client
	operation string
	purger    *redisPurger
}

func RedisFixtureWriterFactory(name *string, operation *string) FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		client, err := redis.ProvideClient(config, logger, *name)
		if err != nil {
			return nil, fmt.Errorf("can not create redis client: %w", err)
		}

		purger, err := newRedisPurger(config, logger, name)
		if err != nil {
			return nil, fmt.Errorf("can not create redis purger: %w", err)
		}

		return NewRedisFixtureWriterWithInterfaces(logger, client, purger, operation), nil
	}
}

func NewRedisFixtureWriterWithInterfaces(logger log.Logger, client redis.Client, purger *redisPurger, operation *string) FixtureWriter {
	return &redisFixtureWriter{
		logger:    logger,
		client:    client,
		purger:    purger,
		operation: *operation,
	}
}

func (d *redisFixtureWriter) Purge(ctx context.Context) error {
	return d.purger.purge(ctx)
}

func (d *redisFixtureWriter) Write(ctx context.Context, fs *FixtureSet) error {
	for _, item := range fs.Fixtures {
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

	d.logger.Info("loaded %d redis fixtures", len(fs.Fixtures))

	return nil
}
