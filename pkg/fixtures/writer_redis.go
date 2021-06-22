package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/redis"
	"time"
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

type redisOpHandler func(client redis.Client, fixture *RedisFixture) error

var redisHandlers = map[string]redisOpHandler{
	RedisOpSet: func(client redis.Client, fixture *RedisFixture) error {
		return client.Set(context.Background(), fixture.Key, fixture.Value, fixture.Expiry)
	},
	RedisOpRpush: func(client redis.Client, fixture *RedisFixture) error {
		_, err := client.RPush(context.Background(), fixture.Key, fixture.Value.([]interface{})...)

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
	return func(config cfg.Config, logger log.Logger) (FixtureWriter, error) {
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

func (d *redisFixtureWriter) Purge() error {
	return d.purger.purge()
}

func (d *redisFixtureWriter) Write(fs *FixtureSet) error {
	for _, item := range fs.Fixtures {
		redisFixture := item.(*RedisFixture)

		handler, ok := redisHandlers[d.operation]

		if !ok {
			return fmt.Errorf("no handler for operation: %s", d.operation)
		}

		err := handler(d.client, redisFixture)

		if err != nil {
			return err
		}
	}

	d.logger.Info("loaded %d redis fixtures", len(fs.Fixtures))

	return nil
}
