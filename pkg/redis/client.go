package redis

import (
	"context"
	"fmt"
	"time"

	baseRedis "github.com/go-redis/redis/v8"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	Nil = baseRedis.Nil
)

type ErrCmder interface {
	Err() error
}

type Z struct {
	Score  float64
	Member interface{}
}

type ZAddArgs struct {
	Key     string
	NX      bool
	XX      bool
	LT      bool
	GT      bool
	Ch      bool
	Incr    bool
	Members []Z
}

type ZRangeArgs struct {
	Key     string
	Start   interface{}
	Stop    interface{}
	ByScore bool
	ByLex   bool
	Rev     bool
	Offset  int64
	Count   int64
}

//go:generate mockery --name Pipeliner
type Pipeliner interface {
	baseRedis.Pipeliner
}

func GetFullyQualifiedKey(appId cfg.AppId, key string) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s", appId.Project, appId.Environment, appId.Family, appId.Group, appId.Application, key)
}

//go:generate mockery --name Client
type Client interface {
	Del(ctx context.Context, keys ...string) (int64, error)
	DBSize(ctx context.Context) (int64, error)
	Exists(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) (bool, error)
	FlushDB(ctx context.Context) (string, error)
	Get(ctx context.Context, key string) (string, error)
	GetDel(ctx context.Context, key string) (interface{}, error)
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)
	MSet(ctx context.Context, pairs ...interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)

	BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error)
	LPop(ctx context.Context, key string) (string, error)
	LLen(ctx context.Context, key string) (int64, error)
	LPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	LRem(ctx context.Context, key string, count int64, value interface{}) (int64, error)
	RPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	RPop(ctx context.Context, key string) (string, error)

	HDel(ctx context.Context, key string, fields ...string) (int64, error)
	HExists(ctx context.Context, key string, field string) (bool, error)
	HGet(ctx context.Context, key string, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HKeys(ctx context.Context, key string) ([]string, error)
	HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error)
	HMSet(ctx context.Context, key string, pairs map[string]interface{}) error
	HSet(ctx context.Context, key string, field string, value interface{}) error
	HSetNX(ctx context.Context, key string, field string, value interface{}) (bool, error)

	SAdd(ctx context.Context, key string, values ...interface{}) (int64, error)
	SCard(ctx context.Context, key string) (int64, error)
	SDiff(ctx context.Context, keys ...string) ([]string, error)
	SDiffStore(ctx context.Context, destination string, keys ...string) (int64, error)
	SInter(ctx context.Context, keys ...string) ([]string, error)
	SInterStore(ctx context.Context, destination string, keys ...string) (int64, error)
	SMembers(ctx context.Context, key string) ([]string, error)
	SIsMember(ctx context.Context, key string, value interface{}) (bool, error)
	SMove(ctx context.Context, sourceKey string, destKey string, member interface{}) (bool, error)
	SPop(ctx context.Context, key string) (string, error)
	SRem(ctx context.Context, key string, values ...interface{}) (int64, error)
	SRandMember(ctx context.Context, key string) (string, error)
	SUnion(ctx context.Context, keys ...string) ([]string, error)
	SUnionStore(ctx context.Context, destination string, keys ...string) (int64, error)

	Decr(ctx context.Context, key string) (int64, error)
	DecrBy(ctx context.Context, key string, amount int64) (int64, error)
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, amount int64) (int64, error)

	PFAdd(ctx context.Context, key string, els ...interface{}) (int64, error)
	PFCount(ctx context.Context, keys ...string) (int64, error)
	PFMerge(ctx context.Context, dest string, keys ...string) (string, error)

	ZAdd(ctx context.Context, key string, score float64, member string) (int64, error)
	ZAddArgs(ctx context.Context, args ZAddArgs) (int64, error)
	ZAddArgsIncr(ctx context.Context, args ZAddArgs) (float64, error)
	ZCard(ctx context.Context, key string) (int64, error)
	ZCount(ctx context.Context, key string, min string, max string) (int64, error)
	ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error)
	ZScore(ctx context.Context, key string, member string) (float64, error)
	ZMScore(ctx context.Context, key string, members ...string) ([]float64, error)
	ZRange(ctx context.Context, key string, start int64, stop int64) ([]string, error)
	ZRangeArgs(ctx context.Context, args ZRangeArgs) ([]string, error)
	ZRangeArgsWithScore(ctx context.Context, args ZRangeArgs) ([]Z, error)
	ZRandMember(ctx context.Context, key string, count int) ([]string, error)
	ZRank(ctx context.Context, key string, member string) (int64, error)
	ZRem(ctx context.Context, key string, members ...string) (int64, error)
	ZRevRank(ctx context.Context, key string, member string) (int64, error)

	IsAlive(ctx context.Context) bool

	Pipeline() Pipeliner
}

type redisClient struct {
	base     baseRedis.Cmdable
	logger   log.Logger
	executor exec.Executor
	settings *Settings
}

func NewClient(config cfg.Config, logger log.Logger, name string) (Client, error) {
	settings := ReadSettings(config, name)

	logger = logger.WithFields(log.Fields{
		"redis": name,
	})

	executor := NewExecutor(logger, settings.BackoffSettings, name)

	if _, ok := dialers[settings.Dialer]; !ok {
		return nil, fmt.Errorf("there is no redis dialer of type %s", settings.Dialer)
	}

	dialer := dialers[settings.Dialer](logger, settings)
	baseClient := baseRedis.NewClient(&baseRedis.Options{
		Dialer: dialer,
	})

	return NewClientWithInterfaces(logger, baseClient, executor, settings), nil
}

func NewClientWithInterfaces(logger log.Logger, baseRedis baseRedis.Cmdable, executor exec.Executor, settings *Settings) Client {
	return &redisClient{
		logger:   logger,
		base:     baseRedis,
		executor: executor,
		settings: settings,
	}
}

func (c *redisClient) GetBaseClient(ctx context.Context) baseRedis.Cmdable {
	c.base.Exists(ctx)

	return c.base
}

func (c *redisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.Exists(ctx, keys...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) FlushDB(ctx context.Context) (string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.FlushDB(ctx)
	})

	return cmd.(*baseRedis.StatusCmd).Val(), err
}

func (c *redisClient) DBSize(ctx context.Context) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.DBSize(ctx)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	_, err := c.execute(ctx, func() ErrCmder {
		return c.base.Set(ctx, key, value, expiration)
	})

	return err
}

func (c *redisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.SetNX(ctx, key, value, expiration)
	})

	val := res.(*baseRedis.BoolCmd).Val()

	return val, err
}

func (c *redisClient) MSet(ctx context.Context, pairs ...interface{}) error {
	_, err := c.execute(ctx, func() ErrCmder {
		return c.base.MSet(ctx, pairs...)
	})

	return err
}

func (c *redisClient) HMSet(ctx context.Context, key string, pairs map[string]interface{}) error {
	_, err := c.execute(ctx, func() ErrCmder {
		return c.base.HMSet(ctx, key, pairs)
	})

	return err
}

func (c *redisClient) Get(ctx context.Context, key string) (string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.Get(ctx, key)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.MGet(ctx, keys...)
	})

	return cmd.(*baseRedis.SliceCmd).Val(), err
}

func (c *redisClient) HMGet(ctx context.Context, key string, fields ...string) ([]interface{}, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.HMGet(ctx, key, fields...)
	})

	return cmd.(*baseRedis.SliceCmd).Val(), err
}

func (c *redisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.Del(ctx, keys...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.BLPop(ctx, timeout, keys...)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) LPop(ctx context.Context, key string) (string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.LPop(ctx, key)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) LLen(ctx context.Context, key string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.LLen(ctx, key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.LPush(ctx, key, values...)
	})

	val := res.(*baseRedis.IntCmd).Val()

	return val, err
}

func (c *redisClient) LRem(ctx context.Context, key string, count int64, value interface{}) (int64, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.LRem(ctx, key, count, value)
	})

	val := res.(*baseRedis.IntCmd).Val()

	return val, err
}

func (c *redisClient) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.RPush(ctx, key, values...)
	})

	val := res.(*baseRedis.IntCmd).Val()

	return val, err
}

func (c *redisClient) RPop(ctx context.Context, key string) (string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.RPop(ctx, key)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) HExists(ctx context.Context, key, field string) (bool, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.HExists(ctx, key, field)
	})

	return cmd.(*baseRedis.BoolCmd).Val(), err
}

func (c *redisClient) HKeys(ctx context.Context, key string) ([]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.HKeys(ctx, key)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) HGet(ctx context.Context, key, field string) (string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.HGet(ctx, key, field)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) HSet(ctx context.Context, key, field string, value interface{}) error {
	_, err := c.execute(ctx, func() ErrCmder {
		return c.base.HSet(ctx, key, field, value)
	})

	return err
}

func (c *redisClient) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.HDel(ctx, key, fields...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.HGetAll(ctx, key)
	})

	return cmd.(*baseRedis.StringStringMapCmd).Val(), err
}

func (c *redisClient) HSetNX(ctx context.Context, key, field string, value interface{}) (bool, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.HSetNX(ctx, key, field, value)
	})

	val := res.(*baseRedis.BoolCmd).Val()

	return val, err
}

func (c *redisClient) SAdd(ctx context.Context, key string, values ...interface{}) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SAdd(ctx, key, values...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) SCard(ctx context.Context, key string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SCard(ctx, key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) SDiff(ctx context.Context, keys ...string) ([]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SDiff(ctx, keys...)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) SDiffStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SDiffStore(ctx, destination, keys...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) SInter(ctx context.Context, keys ...string) ([]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SInter(ctx, keys...)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) SInterStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SInterStore(ctx, destination, keys...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.SMembers(ctx, key)
	})

	val := res.(*baseRedis.StringSliceCmd).Val()

	return val, err
}

func (c *redisClient) SIsMember(ctx context.Context, key string, value interface{}) (bool, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.SIsMember(ctx, key, value)
	})

	val := res.(*baseRedis.BoolCmd).Val()

	return val, err
}

func (c *redisClient) SMove(ctx context.Context, sourceKey string, destKey string, member interface{}) (bool, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.SMove(ctx, sourceKey, destKey, member)
	})

	val := res.(*baseRedis.BoolCmd).Val()

	return val, err
}

func (c *redisClient) SPop(ctx context.Context, key string) (string, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.SPop(ctx, key)
	})

	val := res.(*baseRedis.StringCmd).Val()

	return val, err
}

func (c *redisClient) SRem(ctx context.Context, key string, values ...interface{}) (int64, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.SRem(ctx, key, values...)
	})

	val := res.(*baseRedis.IntCmd).Val()

	return val, err
}

func (c *redisClient) SRandMember(ctx context.Context, key string) (string, error) {
	res, err := c.execute(ctx, func() ErrCmder {
		return c.base.SRandMember(ctx, key)
	})

	val := res.(*baseRedis.StringCmd).Val()

	return val, err
}

func (c *redisClient) SUnion(ctx context.Context, keys ...string) ([]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SUnion(ctx, keys...)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) SUnionStore(ctx context.Context, destination string, keys ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.SUnionStore(ctx, destination, keys...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) Incr(ctx context.Context, key string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.Incr(ctx, key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) IncrBy(ctx context.Context, key string, amount int64) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.IncrBy(ctx, key, amount)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) Decr(ctx context.Context, key string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.Decr(ctx, key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) DecrBy(ctx context.Context, key string, amount int64) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.DecrBy(ctx, key, amount)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) Expire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.Expire(ctx, key, ttl)
	})

	return cmd.(*baseRedis.BoolCmd).Val(), err
}

func (c *redisClient) PFAdd(ctx context.Context, key string, els ...interface{}) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.PFAdd(ctx, key, els...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) PFCount(ctx context.Context, keys ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.PFCount(ctx, keys...)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) PFMerge(ctx context.Context, dest string, keys ...string) (string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.PFMerge(ctx, dest, keys...)
	})

	return cmd.(*baseRedis.StatusCmd).Val(), err
}

func (c *redisClient) ZAdd(ctx context.Context, key string, score float64, member string) (int64, error) {
	args := ZAddArgs{
		Key: key,
		Members: []Z{
			{
				Member: member,
				Score:  score,
			},
		},
	}

	return c.ZAddArgs(ctx, args)
}

func (c *redisClient) ZAddArgs(ctx context.Context, args ZAddArgs) (int64, error) {
	zAddArgs := c.toGoRedisZAddArgs(args)

	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZAddArgs(ctx, args.Key, zAddArgs)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) ZAddArgsIncr(ctx context.Context, args ZAddArgs) (float64, error) {
	zAddArgs := c.toGoRedisZAddArgs(args)

	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZAddArgsIncr(ctx, args.Key, zAddArgs)
	})

	return cmd.(*baseRedis.FloatCmd).Val(), err
}

func (c *redisClient) ZCard(ctx context.Context, key string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZCard(ctx, key)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) ZCount(ctx context.Context, key string, min string, max string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZCount(ctx, key, min, max)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZIncrBy(ctx, key, increment, member)
	})

	return cmd.(*baseRedis.FloatCmd).Val(), err
}

func (c *redisClient) ZScore(ctx context.Context, key string, member string) (float64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZScore(ctx, key, member)
	})

	return cmd.(*baseRedis.FloatCmd).Val(), err
}

func (c *redisClient) ZMScore(ctx context.Context, key string, members ...string) ([]float64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZMScore(ctx, key, members...)
	})

	return cmd.(*baseRedis.FloatSliceCmd).Val(), err
}

func (c *redisClient) ZRange(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	return c.ZRangeArgs(ctx, ZRangeArgs{
		Key:   key,
		Start: start,
		Stop:  stop,
	})
}

func (c *redisClient) ZRangeArgs(ctx context.Context, args ZRangeArgs) ([]string, error) {
	zRangeArgs := baseRedis.ZRangeArgs(args)

	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZRangeArgs(ctx, zRangeArgs)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) ZRangeArgsWithScore(ctx context.Context, args ZRangeArgs) ([]Z, error) {
	zRangeArgs := baseRedis.ZRangeArgs(args)

	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZRangeArgsWithScores(ctx, zRangeArgs)
	})

	zs := cmd.(*baseRedis.ZSliceCmd).Val()
	members := c.toGosolineZs(zs)

	return members, err
}

func (c *redisClient) ZRandMember(ctx context.Context, key string, count int) ([]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZRandMember(ctx, key, count, false)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) ZRank(ctx context.Context, key string, member string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZRank(ctx, key, member)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) ZRem(ctx context.Context, key string, members ...string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZRem(ctx, key, members)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) ZRevRange(ctx context.Context, key string, start int64, stop int64) ([]string, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZRevRange(ctx, key, start, stop)
	})

	return cmd.(*baseRedis.StringSliceCmd).Val(), err
}

func (c *redisClient) ZRevRank(ctx context.Context, key string, member string) (int64, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.ZRevRank(ctx, key, member)
	})

	return cmd.(*baseRedis.IntCmd).Val(), err
}

func (c *redisClient) IsAlive(ctx context.Context) bool {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.Ping(ctx)
	})

	alive := cmd.(*baseRedis.StatusCmd).Val() == "PONG"

	return alive && err == nil
}

func (c *redisClient) GetDel(ctx context.Context, key string) (interface{}, error) {
	cmd, err := c.execute(ctx, func() ErrCmder {
		return c.base.GetDel(ctx, key)
	})

	return cmd.(*baseRedis.StringCmd).Val(), err
}

func (c *redisClient) Pipeline() Pipeliner {
	return c.base.Pipeline()
}

func (c *redisClient) execute(ctx context.Context, wrappedCmd func() ErrCmder) (interface{}, error) {
	return c.executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		cmder := wrappedCmd()

		return cmder, cmder.Err()
	})
}

func (c *redisClient) toGosolineZs(zs []baseRedis.Z) []Z {
	result := make([]Z, len(zs))
	for i := range zs {
		result[i] = Z(zs[i])
	}

	return result
}

func (c *redisClient) toGoRedisZs(zs []Z) []baseRedis.Z {
	result := make([]baseRedis.Z, len(zs))
	for i := range zs {
		result[i] = baseRedis.Z(zs[i])
	}

	return result
}

func (c *redisClient) toGoRedisZAddArgs(args ZAddArgs) baseRedis.ZAddArgs {
	zs := c.toGoRedisZs(args.Members)
	return baseRedis.ZAddArgs{
		NX:      args.NX,
		XX:      args.XX,
		LT:      args.LT,
		GT:      args.GT,
		Ch:      args.Ch,
		Members: zs,
	}
}
