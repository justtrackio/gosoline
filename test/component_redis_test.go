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

	pkgTest.Boot("test_configs/config.redis.test.yml")
	defer pkgTest.Shutdown()

	client := redis.NewClient(&redis.Options{
		Addr: "172.17.0.1:6381",
	})
	pong, err := client.Ping().Result()

	assert.NoError(t, err)
	assert.Equal(t, "PONG", pong)
	assert.Len(t, pong, 4)
}
