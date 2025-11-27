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
	if len(fixtures) == 0 {
		return nil
	}

	switch d.operation {
	case RedisOpSet:
		return d.writeSet(ctx, fixtures)
	case RedisOpRpush:
		return d.writeRpush(ctx, fixtures)
	default:
		return fmt.Errorf("no handler for operation: %s", d.operation)
	}
}

func (d *redisFixtureWriter) writeSet(ctx context.Context, fixtures []any) error {
	// Group fixtures by TTL to determine batching strategy
	// Fixtures with no TTL (0) can use MSet, others need individual Set calls
	noTTLPairs := make([]any, 0)
	withTTL := make([]*RedisFixture, 0)

	for _, item := range fixtures {
		fixture := item.(*RedisFixture)
		if fixture.Expiry == 0 {
			noTTLPairs = append(noTTLPairs, fixture.Key, fixture.Value)
		} else {
			withTTL = append(withTTL, fixture)
		}
	}

	// Batch insert fixtures without TTL using MSet
	if len(noTTLPairs) > 0 {
		if err := d.client.MSet(ctx, noTTLPairs...); err != nil {
			return fmt.Errorf("can not batch set redis fixtures: %w", err)
		}
	}

	// Handle fixtures with TTL individually (MSet doesn't support TTL)
	for _, fixture := range withTTL {
		if err := d.client.Set(ctx, fixture.Key, fixture.Value, fixture.Expiry); err != nil {
			return fmt.Errorf("can not set redis fixture %s: %w", fixture.Key, err)
		}
	}

	d.logger.Info(ctx, "loaded %d redis SET fixtures", len(fixtures))

	return nil
}

func (d *redisFixtureWriter) writeRpush(ctx context.Context, fixtures []any) error {
	// RPUSH operates on different keys with different values, so we can't easily batch
	// Keep individual calls but could use pipelining for performance
	for _, item := range fixtures {
		fixture := item.(*RedisFixture)
		if _, err := d.client.RPush(ctx, fixture.Key, fixture.Value.([]any)...); err != nil {
			return fmt.Errorf("can not rpush redis fixture %s: %w", fixture.Key, err)
		}
	}

	d.logger.Info(ctx, "loaded %d redis RPUSH fixtures", len(fixtures))

	return nil
}
