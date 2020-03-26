//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_redis(t *testing.T) {
	setup(t)

	mocks, err := pkgTest.Boot("test_configs/config.redis.test.yml")
	defer func() {
		if mocks != nil {
			mocks.Shutdown()
		}
	}()

	if err != nil {
		assert.Fail(t, "failed to boot mocks")

		return
	}

	client := mocks.ProvideRedisClient("redis")
	pong, err := client.Ping().Result()

	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)
	assert.Len(t, pong, 4)
}
