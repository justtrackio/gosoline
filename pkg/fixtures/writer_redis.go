package fixtures

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
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
		return client.Set(fixture.Key, fixture.Value, fixture.Expiry)
	},
	RedisOpRpush: func(client redis.Client, fixture *RedisFixture) error {
		_, err := client.RPush(fixture.Key, fixture.Value.([]interface{})...)

		return err
	},
}

type redisFixtureWriter struct {
	logger    mon.Logger
	client    redis.Client
	operation string
}

func RedisFixtureWriterFactory(name *string, operation *string) FixtureWriterFactory {
	return func(config cfg.Config, logger mon.Logger) FixtureWriter {
		client := redis.GetClient(config, logger, *name)

		return NewRedisFixtureWriterWithInterfaces(logger, client, operation)
	}
}

func NewRedisFixtureWriterWithInterfaces(logger mon.Logger, client redis.Client, operation *string) FixtureWriter {
	return &redisFixtureWriter{
		logger:    logger,
		client:    client,
		operation: *operation,
	}
}

func (d *redisFixtureWriter) Purge() error {
	return nil
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

	d.logger.Infof("loaded %d redis fixtures", len(fs.Fixtures))

	return nil
}
