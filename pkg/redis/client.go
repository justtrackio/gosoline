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
	Expire(key string, ttl time.Duration) (bool, error)
	FlushDB() (string, error)
	Set(key string, value interface{}, ttl time.Duration) error
	SetNX(key string, value interface{}, ttl time.Duration) (bool, error)
	MSet(pairs ...interface{}) error
	Get(key string) (string, error)
	MGet(keys ...string) ([]interface{}, error)
	Del(key string) (int64, error)

	BLPop(timeout time.Duration, keys ...string) ([]string, error)
	LPop(key string) (string, error)
	LLen(key string) (int64, error)
	RPush(key string, values ...interface{}) (int64, error)

	HExists(key string, field string) (bool, error)
	HKeys(key string) ([]string, error)
	HGet(key string, field string) (string, error)
	HSet(key string, field string, value interface{}) error
	HMGet(key string, fields ...string) ([]interface{}, error)
	HMSet(key string, pairs map[string]interface{}) error
	HSetNX(key string, field string, value interface{}) (bool, error)

	Incr(key string) (int64, error)
	IncrBy(key string, amount int64) (int64, error)
	Decr(key string) (int64, error)
	DecrBy(key string, amount int64) (int64, error)

	IsAlive() bool

	Pipeline() baseRedis.Pipeliner
}

type redisClient struct {
	base     baseRedis.Cmdable
	logger   mon.Logger
	metric   mon.MetricWriter
	settings *Settings
}

func NewClient(config cfg.Config, logger mon.Logger, name string) Client {
	settings := ReadSettings(config, name)

	defaults := getMetricDefaults(name)
	metric := mon.NewMetricDaemonWriter(defaults...)

	logger = logger.WithFields(mon.Fields{
		"redis": name,
	})

	if _, ok := dialers[settings.Dialer]; !ok {
		logger.Fatalf(fmt.Errorf("dialer not found"), "there is no redis dialer of type %s", settings.Dialer)
		return nil
	}

	dialer := dialers[settings.Dialer](logger, settings)
	baseClient := baseRedis.NewClient(&baseRedis.Options{
		Dialer: dialer,
	})

	redisClient := &redisClient{
		logger:   logger,
		metric:   metric,
		settings: settings,
		base:     baseClient,
	}

	return redisClient
}

func NewClientWithInterfaces(logger mon.Logger, baseRedis baseRedis.Cmdable, writer mon.MetricWriter, settings *Settings) Client {
	return &redisClient{
		logger:   logger,
		base:     baseRedis,
		metric:   writer,
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

func (c *redisClient) FlushDB() (string, error) {
	return c.base.FlushDB().Result()
}

func (c *redisClient) Set(key string, value interface{}, expiration time.Duration) error {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
		cmd := c.base.Set(key, value, expiration)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.StatusCmd).Err()
}

func (c *redisClient) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
		cmd := c.base.SetNX(key, value, expiration)

		return cmd, cmd.Err()
	}).(*baseRedis.BoolCmd)

	return res.Val(), res.Err()
}

func (c *redisClient) MSet(pairs ...interface{}) error {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
		cmd := c.base.MSet(pairs...)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.StatusCmd).Err()
}

func (c *redisClient) HMSet(key string, pairs map[string]interface{}) error {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
		cmd := c.base.HMSet(key, pairs)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.StatusCmd).Err()
}

func (c *redisClient) Get(key string) (string, error) {
	return c.base.Get(key).Result()
}

func (c *redisClient) MGet(keys ...string) ([]interface{}, error) {
	return c.base.MGet(keys...).Result()
}

func (c *redisClient) HMGet(key string, fields ...string) ([]interface{}, error) {
	return c.base.HMGet(key, fields...).Result()
}

func (c *redisClient) Del(key string) (int64, error) {
	return c.base.Del(key).Result()
}

func (c *redisClient) BLPop(timeout time.Duration, keys ...string) ([]string, error) {
	return c.base.BLPop(timeout, keys...).Result()
}

func (c *redisClient) LPop(key string) (string, error) {
	return c.base.LPop(key).Result()
}

func (c *redisClient) LLen(key string) (int64, error) {
	return c.base.LLen(key).Result()
}

func (c *redisClient) RPush(key string, values ...interface{}) (int64, error) {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
		cmd := c.base.RPush(key, values...)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.IntCmd).Result()
}

func (c *redisClient) HExists(key, field string) (bool, error) {
	return c.base.HExists(key, field).Result()
}

func (c *redisClient) HKeys(key string) ([]string, error) {
	return c.base.HKeys(key).Result()
}

func (c *redisClient) HGet(key, field string) (string, error) {
	return c.base.HGet(key, field).Result()
}

func (c *redisClient) HSet(key, field string, value interface{}) error {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
		cmd := c.base.HSet(key, field, value)

		return cmd, cmd.Err()
	})

	return res.(*baseRedis.BoolCmd).Err()
}

func (c *redisClient) HSetNX(key, field string, value interface{}) (bool, error) {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
		cmd := c.base.HSetNX(key, field, value)

		return cmd, cmd.Err()
	}).(*baseRedis.BoolCmd)

	return res.Val(), res.Err()
}

func (c *redisClient) Incr(key string) (int64, error) {
	return c.base.Incr(key).Result()
}

func (c *redisClient) IncrBy(key string, amount int64) (int64, error) {
	return c.base.IncrBy(key, amount).Result()
}

func (c *redisClient) Decr(key string) (int64, error) {
	return c.base.Decr(key).Result()
}

func (c *redisClient) DecrBy(key string, amount int64) (int64, error) {
	return c.base.DecrBy(key, amount).Result()
}

func (c *redisClient) Expire(key string, ttl time.Duration) (bool, error) {
	return c.base.Expire(key, ttl).Result()
}

func (c *redisClient) IsAlive() bool {
	return c.base.Ping().Err() == nil
}

func (c *redisClient) Pipeline() baseRedis.Pipeliner {
	return c.base.Pipeline()
}

func (c *redisClient) attemptPreventingFailuresByBackoff(wrappedCmd func() (interface{}, error)) interface{} {
	backOffSettings := c.settings.BackoffSettings

	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = backOffSettings.InitialInterval
	backoffConfig.MaxInterval = backOffSettings.MaxInterval
	backoffConfig.MaxElapsedTime = backOffSettings.MaxElapsedTime
	backoffConfig.Multiplier = backOffSettings.Multiplier
	backoffConfig.RandomizationFactor = backOffSettings.RandomizationFactor

	var res interface{}
	var err error

	notify := func(err error, duration time.Duration) {
		c.logger.WithFields(mon.Fields{
			"name":     c.settings.Name,
			"err":      err,
			"duration": duration,
		}).Infof("redis %s is blocking due to error %s", c.settings.Name, err.Error())

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
		if err == nil {
			return nil
		}

		if strings.HasPrefix(err.Error(), "OOM") {
			return err
		}

		return backoff.Permanent(err)
	}

	err = backoff.RetryNotify(operation, backoffConfig, notify)

	return res
}

func getMetricDefaults(name string) []*mon.MetricDatum {
	return []*mon.MetricDatum{
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
}
