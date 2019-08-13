package redis

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/cenkalti/backoff"
	baseRedis "github.com/go-redis/redis"
	"strings"
	"time"
)

const (
	Nil                      = baseRedis.Nil
	metricClientBackoffCount = "RedisClientBackoffCount"
)

func GetFullyQualifiedKey(appId cfg.AppId, key string) string {
	return fmt.Sprintf("%v-%v-%v-%v-%v", appId.Project, appId.Environment, appId.Family, appId.Application, key)
}

//go:generate mockery -name Client
type Client interface {
	Exists(keys ...string) (int64, error)
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string) (string, error)
	Del(key string) (int64, error)

	BLPop(timeout time.Duration, keys ...string) ([]string, error)
	LLen(key string) (int64, error)
	RPush(key string, values ...interface{}) (int64, error)

	HGet(key, field string) (string, error)
	HSet(key, field string, value interface{}) error

	Pipeline() baseRedis.Pipeliner
}

type redisClient struct {
	base   baseRedis.Cmdable
	metric mon.MetricWriter

	name string
}

func NewRedisClient(client baseRedis.Cmdable, name string) Client {
	defaults := mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricClientBackoffCount,
			Dimensions: map[string]string{
				"Redis": name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}

	metric := mon.NewMetricDaemonWriter(defaults...)

	return &redisClient{
		base:   client,
		metric: metric,
		name:   name,
	}
}

func (c *redisClient) GetBaseClient() baseRedis.Cmdable {
	c.base.Exists()

	return c.base
}

func (c *redisClient) Exists(keys ...string) (int64, error) {
	return c.base.Exists(keys...).Result()
}

func (c *redisClient) Set(key string, value interface{}, expiration time.Duration) error {
	res := c.preventOOMByBackoff(func() (interface{}, error) {
		cmd := c.base.Set(key, value, expiration)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.StatusCmd).Err()
}

func (c *redisClient) Get(key string) (string, error) {
	return c.base.Get(key).Result()
}

func (c *redisClient) Del(key string) (int64, error) {
	return c.base.Del(key).Result()
}

func (c *redisClient) BLPop(timeout time.Duration, keys ...string) ([]string, error) {
	return c.base.BLPop(timeout, keys...).Result()
}

func (c *redisClient) LLen(key string) (int64, error) {
	return c.base.LLen(key).Result()
}

func (c *redisClient) RPush(key string, values ...interface{}) (int64, error) {
	res := c.preventOOMByBackoff(func() (interface{}, error) {
		cmd := c.base.RPush(key, values...)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.IntCmd).Result()
}

func (c *redisClient) HGet(key, field string) (string, error) {
	return c.base.HGet(key, field).Result()
}

func (c *redisClient) HSet(key, field string, value interface{}) error {
	res := c.preventOOMByBackoff(func() (interface{}, error) {
		cmd := c.base.HSet(key, field, value)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.BoolCmd).Err()
}

func (c *redisClient) Pipeline() baseRedis.Pipeliner {
	return c.base.Pipeline()
}

func (c *redisClient) preventOOMByBackoff(wrappedCmd func() (interface{}, error)) interface{} {
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = 200 * time.Millisecond
	backoffConfig.MaxInterval = 30 * time.Second
	backoffConfig.Multiplier = 3
	backoffConfig.RandomizationFactor = 0.2

	var res interface{}
	var err error

	notify := func(error, time.Duration) {
		c.metric.WriteOne(&mon.MetricDatum{
			MetricName: metricClientBackoffCount,
			Value:      1.0,
			Dimensions: map[string]string{
				"Redis": c.name,
			},
		})
	}

	operation := func() error {
		res, err = wrappedCmd()

		if err != nil && !strings.HasPrefix(err.Error(), "OOM") {
			err = backoff.Permanent(err)
		}

		return err
	}

	// No further error handling as the wrapped redis error is handled elsewhere
	err = backoff.RetryNotify(operation, backoffConfig, notify)

	return res
}
