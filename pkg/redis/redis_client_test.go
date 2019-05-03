package redis_test

import (
	"github.com/alicebob/miniredis"
	"github.com/applike/gosoline/pkg/redis"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var err error
var s *miniredis.Miniredis
var c redis.Client

func TestRedisBLPop(t *testing.T) {
	buildClient()

	if _, err := s.Lpush("list", "value"); err != nil {
		panic(err)
	}

	res, err := c.BLPop(1*time.Second, "list")

	assert.Nil(t, err, "there should be no error on blpop")
	assert.Equal(t, "value", res[1])
}

func TestRedisDel(t *testing.T) {
	buildClient()

	count, err := c.Del("test")
	assert.Nil(t, err, "there should be no error on Del")
	assert.Equal(t, 0, int(count))

	var ttl time.Duration
	err = c.Set("key", "value", ttl)
	assert.Nil(t, err, "there should be no error on Del")

	count, err = c.Del("key")
	assert.Nil(t, err, "there should be no error on Del")
	assert.Equal(t, 1, int(count))
}

func TestRedisLLen(t *testing.T) {
	buildClient()

	for i := 0; i < 3; i++ {
		if _, err := s.Lpush("list", "value"); err != nil {
			panic(err)
		}
	}

	res, err := c.LLen("list")

	assert.Nil(t, err, "there should be no error on LLen")
	assert.Equal(t, int64(3), res)
}

func TestRedisRPush(t *testing.T) {
	buildClient()

	count, err := c.RPush("list", "v1", "v2", "v3")
	assert.Nil(t, err, "there should be no error on RPush")
	assert.Equal(t, int64(3), count)
}

func TestRedisSet(t *testing.T) {
	buildClient()

	var ttl time.Duration
	err := c.Set("key", "value", ttl)
	assert.Nil(t, err, "there should be no error on Set")

	ttl, _ = time.ParseDuration("1m")
	err = c.Set("key", "value", ttl)
	assert.Nil(t, err, "there should be no error on Set with expiration date")
}

func buildClient() (*miniredis.Miniredis, redis.Client) {
	if s != nil {
		s.FlushAll()
		return s, c
	}

	s, err = miniredis.Run()
	if err != nil {
		panic(err)
	}

	c = redis.GetClientWithAddress(s.Addr())

	return s, c
}
