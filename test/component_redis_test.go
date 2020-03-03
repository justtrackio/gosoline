//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_redis(t *testing.T) {
	setup(t)

	mocks := pkgTest.Boot("test_configs/config.redis.test.yml")
	defer mocks.Shutdown()

	client := mocks.ProvideClient("redis", "redis").(*redis.Client)
	pong, err := client.Ping().Result()

	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)
	assert.Len(t, pong, 4)
}
