//go:build integration

package env_test

import (
	"context"
	"net"
	"strconv"
	"testing"

	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/require"
)

func TestExternalRedisDatabaseAllocation(t *testing.T) {
	owner, err := env.NewEnvironment(t, env.WithConfigMap(map[string]any{
		"app.env": "test", "app.name": "redis-test",
		"test.auto_detect.enabled":      false,
		"test.components.redis.default": map[string]any{},
	}))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, owner.Stop()) })
	host, port, err := net.SplitHostPort(owner.Redis("default").Address())
	require.NoError(t, err)
	portNumber, err := strconv.Atoi(port)
	require.NoError(t, err)
	newExternal := func(db int) (*env.Environment, error) {
		return env.NewEnvironment(t, env.WithConfigMap(map[string]any{
			"app.env": "test", "app.name": "redis-test",
			"test.auto_detect.enabled":              false,
			"test.container_manager.runner_type":    "external",
			"test.components.redis.default.host":    host,
			"test.components.redis.default.port":    portNumber,
			"test.components.redis.default.auto_db": true,
			"test.components.redis.default.db":      db,
		}))
	}
	first, err := newExternal(0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, first.Stop()) })
	second, err := newExternal(0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, second.Stop()) })
	firstClient := first.Redis("default").Client()
	secondClient := second.Redis("default").Client()
	require.NotEqual(t, firstClient.Options().DB, secondClient.Options().DB)
	require.Positive(t, firstClient.Options().DB)
	ctx := context.Background()
	require.NoError(t, firstClient.Set(ctx, "same-key", "first", 0).Err())
	require.NoError(t, secondClient.Set(ctx, "same-key", "second", 0).Err())
	require.NoError(t, firstClient.FlushDB(ctx).Err())
	value, err := secondClient.Get(ctx, "same-key").Result()
	require.NoError(t, err)
	require.Equal(t, "second", value)

	explicit, err := newExternal(10)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, explicit.Stop()) })
	require.Equal(t, 10, explicit.Redis("default").Client().Options().DB)

	ownerClient := owner.Redis("default").Client()
	databases, err := ownerClient.ConfigGet(ctx, "databases").Result()
	require.NoError(t, err)
	count, err := strconv.Atoi(databases["databases"])
	require.NoError(t, err)
	require.NoError(t, ownerClient.Set(ctx, "gosoline:test:next-db", count-1, 0).Err())
	failed, err := newExternal(0)
	require.ErrorContains(t, err, "Redis database allocation exhausted")
	if failed != nil {
		require.NoError(t, failed.Stop())
	}
	value, err = secondClient.Get(ctx, "same-key").Result()
	require.NoError(t, err)
	require.Equal(t, "second", value)
}
