//+build integration

package test_test

import (
	pkgTest "github.com/applike/gosoline/pkg/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_redis(t *testing.T) {
	setup(t)

	pkgTest.Boot("test_configs/config.redis.test.yml")
	defer pkgTest.Shutdown()

	client := pkgTest.ProvideRedisClient("redis")
	pong, err := client.Ping().Result()

	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)
	assert.Len(t, pong, 4)
}
