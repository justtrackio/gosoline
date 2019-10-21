package redis

import (
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/cenkalti/backoff"
	baseRedis "github.com/go-redis/redis"
	"net"
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
	Exists(...string) (int64, error)
	Set(string, interface{}, time.Duration) error
	Get(string) (string, error)
	Del(string) (int64, error)

	BLPop(time.Duration, ...string) ([]string, error)
	LLen(string) (int64, error)
	RPush(string, ...interface{}) (int64, error)

	HExists(string, string) (bool, error)
	HKeys(string) ([]string, error)
	HGet(string, string) (string, error)
	HSet(string, string, interface{}) error

	IsAlive() bool

	Pipeline() baseRedis.Pipeliner
}

type Settings struct {
	cfg.AppId
	Name            string
	Mode            string
	Address         string
	BackoffSettings BackoffSettings
}

type BackoffSettings struct {
	InitialInterval     time.Duration `cfg:"initial_interval"`
	RandomizationFactor float64       `cfg:"randomization_factor"`
	Multiplier          float64       `cfg:"multiplier"`
	MaxInterval         time.Duration `cfg:"max_interval"`
	MaxElapsedTime      time.Duration `cfg:"max_elapsed_time"`
}

type redisClient struct {
	base     baseRedis.Cmdable
	logger   mon.Logger
	metric   mon.MetricWriter
	settings *Settings
}

func NewRedisClient(logger mon.Logger, redisSettings *Settings) Client {
	defaults := mon.MetricData{
		{
			Priority:   mon.PriorityHigh,
			MetricName: metricClientBackoffCount,
			Dimensions: map[string]string{
				"Redis": redisSettings.Name,
			},
			Unit:  mon.UnitCount,
			Value: 0.0,
		},
	}

	metric := mon.NewMetricDaemonWriter(defaults...)
	logger = logger.WithFields(mon.Fields{
		"redis": redisSettings.Name,
	})

	dialer := dialerLocal(redisSettings)

	if redisSettings.Mode == RedisModeDiscover {
		dialer = dialerDiscovery(redisSettings)
	}

	baseClient := baseRedis.NewClient(&baseRedis.Options{
		Dialer: dialer,
	})

	redisClient := &redisClient{
		logger:   logger,
		metric:   metric,
		settings: redisSettings,
		base:     baseClient,
	}

	return redisClient
}

func NewRedisClientWithInterfaces(baseRedis baseRedis.Cmdable, logger mon.Logger, writer mon.MetricWriter, settings *Settings) Client {
	return &redisClient{
		base:     baseRedis,
		logger:   logger,
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

func (c *redisClient) Set(key string, value interface{}, expiration time.Duration) error {
	res := c.attemptPreventingFailuresByBackoff(func() (interface{}, error) {
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

func dialerDiscovery(settings *Settings) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		address := settings.Address

		if address == "" {
			address = fmt.Sprintf("%s.redis.%s.%s", settings.Name, settings.Environment, settings.Family)
		}

		_, srvs, err := net.LookupSRV("", "", address)

		if err != nil {
			return nil, err
		}

		if len(srvs) != 1 {
			return nil, errors.New(fmt.Sprintf("redis instance count mismatch. there should be exactly one redis instance, found: %v", len(srvs)))
		}

		address = fmt.Sprintf("%v:%v", srvs[0].Target, srvs[0].Port)

		return net.Dial("tcp", address)
	}
}

func dialerLocal(settings *Settings) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		return net.Dial("tcp", settings.Address)
	}
}
