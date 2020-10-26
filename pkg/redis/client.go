package redis

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon"
	baseRedis "github.com/go-redis/redis"
	"time"
)

const (
	Nil = baseRedis.Nil
)

type ErrCmder interface {
	Err() error
}

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
	executor exec.Executor
	settings *Settings
}

func NewClient(config cfg.Config, logger mon.Logger, name string) Client {
	settings := ReadSettings(config, name)

	logger = logger.WithFields(mon.Fields{
		"redis": name,
	})

	executor := NewExecutor(logger, settings.BackoffSettings, name)

	if _, ok := dialers[settings.Dialer]; !ok {
		logger.Fatalf(fmt.Errorf("dialer not found"), "there is no redis dialer of type %s", settings.Dialer)
		return nil
	}

	dialer := dialers[settings.Dialer](logger, settings)
	baseClient := baseRedis.NewClient(&baseRedis.Options{
		Dialer: dialer,
	})

	return NewClientWithInterfaces(logger, baseClient, executor, settings)
}

func NewClientWithInterfaces(logger mon.Logger, baseRedis baseRedis.Cmdable, executor exec.Executor, settings *Settings) Client {
	return &redisClient{
		logger:   logger,
		base:     baseRedis,
		executor: executor,
		settings: settings,
	}
}

func (c *redisClient) GetBaseClient() baseRedis.Cmdable {
	c.base.Exists()

	return c.base
}

func (c *redisClient) Exists(keys ...string) (int64, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.Exists(keys...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) FlushDB() (string, error) {

	cmd, err := c.execute(func() ErrCmder {
		return c.base.FlushDB()
	})

	return cmd.(*baseRedis.StatusCmd).Val(), err
}

func (c *redisClient) Set(key string, value interface{}, expiration time.Duration) error {
	_, err := c.execute(func() ErrCmder {
		return c.base.Set(key, value, expiration)
	})

	return err
}

func (c *redisClient) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	res, err := c.execute(func() ErrCmder {
		return c.base.SetNX(key, value, expiration)
	})

	val := res.(*baseRedis.BoolCmd).Val()

	return val, err
}

func (c *redisClient) MSet(pairs ...interface{}) error {
	_, err := c.execute(func() ErrCmder {
		return c.base.MSet(pairs...)
	})

	return err
}

func (c *redisClient) HMSet(key string, pairs map[string]interface{}) error {
	_, err := c.execute(func() ErrCmder {
		return c.base.HMSet(key, pairs)
	})

	return err
}

func (c *redisClient) Get(key string) (string, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.Get(key)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) MGet(keys ...string) ([]interface{}, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.MGet(keys...)
	})

	return cmd.(*baseRedis.SliceCmd).Val(), err
}

func (c *redisClient) HMGet(key string, fields ...string) ([]interface{}, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.HMGet(key, fields...)
	})

	return cmd.(*baseRedis.SliceCmd).Val(), err
}

func (c *redisClient) Del(key string) (int64, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.Del(key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) BLPop(timeout time.Duration, keys ...string) ([]string, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.BLPop(timeout, keys...)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) LPop(key string) (string, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.LPop(key)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) LLen(key string) (int64, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.LLen(key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) RPush(key string, values ...interface{}) (int64, error) {
	res, err := c.execute(func() ErrCmder {
		return c.base.RPush(key, values...)
	})

	val := res.(*baseRedis.IntCmd).Val()

	return val, err
}

func (c *redisClient) HExists(key, field string) (bool, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.HExists(key, field)
	})

	return cmd.(*baseRedis.BoolCmd).Val(), err
}

func (c *redisClient) HKeys(key string) ([]string, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.HKeys(key)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) HGet(key, field string) (string, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.HGet(key, field)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) HSet(key, field string, value interface{}) error {
	_, err := c.execute(func() ErrCmder {
		return c.base.HSet(key, field, value)
	})

	return err
}

func (c *redisClient) HSetNX(key, field string, value interface{}) (bool, error) {
	res, err := c.execute(func() ErrCmder {
		return c.base.HSetNX(key, field, value)
	})

	val := res.(*baseRedis.BoolCmd).Val()

	return val, err
}

func (c *redisClient) Incr(key string) (int64, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.Incr(key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) IncrBy(key string, amount int64) (int64, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.IncrBy(key, amount)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) Decr(key string) (int64, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.Decr(key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) DecrBy(key string, amount int64) (int64, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.DecrBy(key, amount)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) Expire(key string, ttl time.Duration) (bool, error) {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.Expire(key, ttl)
	})

	return cmd.(*baseRedis.BoolCmd).Val(), err
}

func (c *redisClient) IsAlive() bool {
	cmd, err := c.execute(func() ErrCmder {
		return c.base.Ping()
	})

	alive := cmd.(*baseRedis.StatusCmd).Val() == "PONG"

	return alive && err == nil
}

func (c *redisClient) Pipeline() baseRedis.Pipeliner {
	return c.base.Pipeline()
}

func (c *redisClient) execute(wrappedCmd func() ErrCmder) (interface{}, error) {
	return c.executor.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		cmder := wrappedCmd()

		return cmder, cmder.Err()
	})
}
