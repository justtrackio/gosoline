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

type Settings struct {
	Name    string
	Backoff SettingsBackoff
}

type SettingsBackoff struct {
	InitialInterval     time.Duration
	RandomizationFactor float64
	Multiplier          float64
	MaxInterval         time.Duration
	MaxElapsedTime      time.Duration
}

type redisClient struct {
	base   baseRedis.Cmdable
	logger mon.Logger
	metric mon.MetricWriter

	settings Settings
}

func NewRedisClient(logger mon.Logger, client baseRedis.Cmdable, settings Settings) Client {
	defaults := mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricClientBackoffCount,
			Dimensions: map[string]string{
				"Redis": settings.Name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}

	metric := mon.NewMetricDaemonWriter(defaults...)
	logger = logger.WithFields(mon.Fields{
		"redis": settings.Name,
	})

	return &redisClient{
		base:     client,
		logger:   logger,
		metric:   metric,
		settings: settings,
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
	backOffSettings := c.settings.Backoff

	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = backOffSettings.InitialInterval * time.Second
	backoffConfig.MaxInterval = backOffSettings.MaxInterval * time.Second
	backoffConfig.Multiplier = backOffSettings.Multiplier
	backoffConfig.RandomizationFactor = backOffSettings.RandomizationFactor
	backoffConfig.MaxElapsedTime = backOffSettings.MaxElapsedTime

	var res interface{}
	var err error

	notify := func(error, time.Duration) {
		c.logger.Infof("redis %s is blocking due to server being out of memory", c.settings.Name)
		c.metric.WriteOne(&mon.MetricDatum{
			MetricName: metricClientBackoffCount,
			Value:      1.0,
			Dimensions: map[string]string{
				"Redis": c.settings.Name,
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
