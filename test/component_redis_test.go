//go:build integration
// +build integration

package test_test

import (
	"context"
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_redis(t *testing.T) {
	t.Parallel()
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.redis.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks: %s", err.Error())

		return
	}

	client := mocks.ProvideRedisClient("redis")
	pong, err := client.Ping(context.Background()).Result()

	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)
	assert.Len(t, pong, 4)
}
