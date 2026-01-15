package redis

import (
	"context"
	"time"

	baseRedis "github.com/redis/go-redis/v9"
)

//go:generate go run github.com/vektra/mockery/v2 --name Pipeliner
type Pipeliner interface {
	baseRedis.Pipeliner
}

type prefixedPipeliner struct {
	baseRedis.Pipeliner
	client *redisClient
}

func (p *prefixedPipeliner) Pipeline() baseRedis.Pipeliner {
	return &prefixedPipeliner{
		Pipeliner: p.Pipeliner.Pipeline(),
		client:    p.client,
	}
}

func (p *prefixedPipeliner) TxPipeline() baseRedis.Pipeliner {
	return &prefixedPipeliner{
		Pipeliner: p.Pipeliner.TxPipeline(),
		client:    p.client,
	}
}

// ---------- Key/Value operations ----------

func (p *prefixedPipeliner) Del(ctx context.Context, keys ...string) *baseRedis.IntCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "del")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Del(ctx, prefixedKeys...)
}

func (p *prefixedPipeliner) Exists(ctx context.Context, keys ...string) *baseRedis.IntCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "exists")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Exists(ctx, prefixedKeys...)
}

func (p *prefixedPipeliner) Expire(ctx context.Context, key string, expiration time.Duration) *baseRedis.BoolCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "expire")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Expire(ctx, key, expiration)
}

func (p *prefixedPipeliner) ExpireNX(ctx context.Context, key string, expiration time.Duration) *baseRedis.BoolCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "expire")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ExpireNX(ctx, key, expiration)
}

func (p *prefixedPipeliner) Get(ctx context.Context, key string) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "get")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Get(ctx, key)
}

func (p *prefixedPipeliner) GetDel(ctx context.Context, key string) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "getdel")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.GetDel(ctx, key)
}

func (p *prefixedPipeliner) GetSet(ctx context.Context, key string, value any) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "getset")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.GetSet(ctx, key, value)
}

func (p *prefixedPipeliner) MGet(ctx context.Context, keys ...string) *baseRedis.SliceCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewSliceCmd(ctx, "mget")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.MGet(ctx, prefixedKeys...)
}

func (p *prefixedPipeliner) MSet(ctx context.Context, pairs ...any) *baseRedis.StatusCmd {
	prefixedPairs, err := p.client.prefixInterleavedKeys(pairs...)
	if err != nil {
		cmd := baseRedis.NewStatusCmd(ctx, "mset")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.MSet(ctx, prefixedPairs...)
}

func (p *prefixedPipeliner) Set(ctx context.Context, key string, value any, expiration time.Duration) *baseRedis.StatusCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStatusCmd(ctx, "set")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Set(ctx, key, value, expiration)
}

func (p *prefixedPipeliner) SetNX(ctx context.Context, key string, value any, expiration time.Duration) *baseRedis.BoolCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "setnx")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SetNX(ctx, key, value, expiration)
}

// ---------- List operations ----------

func (p *prefixedPipeliner) BLPop(ctx context.Context, timeout time.Duration, keys ...string) *baseRedis.StringSliceCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "blpop")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.BLPop(ctx, timeout, prefixedKeys...)
}

func (p *prefixedPipeliner) LPop(ctx context.Context, key string) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "lpop")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.LPop(ctx, key)
}

func (p *prefixedPipeliner) LLen(ctx context.Context, key string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "llen")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.LLen(ctx, key)
}

func (p *prefixedPipeliner) LPush(ctx context.Context, key string, values ...any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "lpush")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.LPush(ctx, key, values...)
}

func (p *prefixedPipeliner) LRem(ctx context.Context, key string, count int64, value any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "lrem")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.LRem(ctx, key, count, value)
}

func (p *prefixedPipeliner) RPush(ctx context.Context, key string, values ...any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "rpush")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.RPush(ctx, key, values...)
}

func (p *prefixedPipeliner) RPop(ctx context.Context, key string) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "rpop")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.RPop(ctx, key)
}

// ---------- Hash operations ----------

func (p *prefixedPipeliner) HDel(ctx context.Context, key string, fields ...string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "hdel")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HDel(ctx, key, fields...)
}

func (p *prefixedPipeliner) HExists(ctx context.Context, key, field string) *baseRedis.BoolCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "hexists")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HExists(ctx, key, field)
}

func (p *prefixedPipeliner) HGet(ctx context.Context, key, field string) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "hget")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HGet(ctx, key, field)
}

func (p *prefixedPipeliner) HGetAll(ctx context.Context, key string) *baseRedis.MapStringStringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewMapStringStringCmd(ctx, "hgetall")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HGetAll(ctx, key)
}

func (p *prefixedPipeliner) HKeys(ctx context.Context, key string) *baseRedis.StringSliceCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "hkeys")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HKeys(ctx, key)
}

func (p *prefixedPipeliner) HMGet(ctx context.Context, key string, fields ...string) *baseRedis.SliceCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewSliceCmd(ctx, "hmget")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HMGet(ctx, key, fields...)
}

func (p *prefixedPipeliner) HMSet(ctx context.Context, key string, values ...any) *baseRedis.BoolCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "hmset")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HMSet(ctx, key, values...)
}

func (p *prefixedPipeliner) HSet(ctx context.Context, key string, values ...any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "hset")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HSet(ctx, key, values...)
}

func (p *prefixedPipeliner) HSetNX(ctx context.Context, key, field string, value any) *baseRedis.BoolCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "hsetnx")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.HSetNX(ctx, key, field, value)
}

// ---------- Set operations ----------

func (p *prefixedPipeliner) SAdd(ctx context.Context, key string, members ...any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "sadd")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SAdd(ctx, key, members...)
}

func (p *prefixedPipeliner) SCard(ctx context.Context, key string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "scard")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SCard(ctx, key)
}

func (p *prefixedPipeliner) SDiff(ctx context.Context, keys ...string) *baseRedis.StringSliceCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "sdiff")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SDiff(ctx, prefixedKeys...)
}

func (p *prefixedPipeliner) SDiffStore(ctx context.Context, destination string, keys ...string) *baseRedis.IntCmd {
	allKeys := append([]string{destination}, keys...)
	prefixedKeys, err := p.client.prefixKeys(allKeys...)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "sdiffstore")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SDiffStore(ctx, prefixedKeys[0], prefixedKeys[1:]...)
}

func (p *prefixedPipeliner) SInter(ctx context.Context, keys ...string) *baseRedis.StringSliceCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "sinter")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SInter(ctx, prefixedKeys...)
}

func (p *prefixedPipeliner) SInterStore(ctx context.Context, destination string, keys ...string) *baseRedis.IntCmd {
	allKeys := append([]string{destination}, keys...)
	prefixedKeys, err := p.client.prefixKeys(allKeys...)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "sinterstore")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SInterStore(ctx, prefixedKeys[0], prefixedKeys[1:]...)
}

func (p *prefixedPipeliner) SMembers(ctx context.Context, key string) *baseRedis.StringSliceCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "smembers")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SMembers(ctx, key)
}

func (p *prefixedPipeliner) SIsMember(ctx context.Context, key string, member any) *baseRedis.BoolCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "sismember")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SIsMember(ctx, key, member)
}

func (p *prefixedPipeliner) SMove(ctx context.Context, source, destination string, member any) *baseRedis.BoolCmd {
	prefixedKeys, err := p.client.prefixKeys(source, destination)
	if err != nil {
		cmd := baseRedis.NewBoolCmd(ctx, "smove")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SMove(ctx, prefixedKeys[0], prefixedKeys[1], member)
}

func (p *prefixedPipeliner) SPop(ctx context.Context, key string) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "spop")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SPop(ctx, key)
}

func (p *prefixedPipeliner) SRem(ctx context.Context, key string, members ...any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "srem")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SRem(ctx, key, members...)
}

func (p *prefixedPipeliner) SRandMember(ctx context.Context, key string) *baseRedis.StringCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringCmd(ctx, "srandmember")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SRandMember(ctx, key)
}

func (p *prefixedPipeliner) SUnion(ctx context.Context, keys ...string) *baseRedis.StringSliceCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "sunion")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SUnion(ctx, prefixedKeys...)
}

func (p *prefixedPipeliner) SUnionStore(ctx context.Context, destination string, keys ...string) *baseRedis.IntCmd {
	allKeys := append([]string{destination}, keys...)
	prefixedKeys, err := p.client.prefixKeys(allKeys...)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "sunionstore")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.SUnionStore(ctx, prefixedKeys[0], prefixedKeys[1:]...)
}

// ---------- Counter operations ----------

func (p *prefixedPipeliner) Decr(ctx context.Context, key string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "decr")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Decr(ctx, key)
}

func (p *prefixedPipeliner) DecrBy(ctx context.Context, key string, decrement int64) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "decrby")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.DecrBy(ctx, key, decrement)
}

func (p *prefixedPipeliner) Incr(ctx context.Context, key string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "incr")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Incr(ctx, key)
}

func (p *prefixedPipeliner) IncrBy(ctx context.Context, key string, value int64) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "incrby")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.IncrBy(ctx, key, value)
}

func (p *prefixedPipeliner) TTL(ctx context.Context, key string) *baseRedis.DurationCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewDurationCmd(ctx, time.Duration(0), "ttl")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.TTL(ctx, key)
}

// ---------- HyperLogLog operations ----------

func (p *prefixedPipeliner) PFAdd(ctx context.Context, key string, els ...any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "pfadd")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.PFAdd(ctx, key, els...)
}

func (p *prefixedPipeliner) PFCount(ctx context.Context, keys ...string) *baseRedis.IntCmd {
	prefixedKeys, err := p.client.prefixKeys(keys...)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "pfcount")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.PFCount(ctx, prefixedKeys...)
}

func (p *prefixedPipeliner) PFMerge(ctx context.Context, dest string, keys ...string) *baseRedis.StatusCmd {
	allKeys := append([]string{dest}, keys...)
	prefixedKeys, err := p.client.prefixKeys(allKeys...)
	if err != nil {
		cmd := baseRedis.NewStatusCmd(ctx, "pfmerge")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.PFMerge(ctx, prefixedKeys[0], prefixedKeys[1:]...)
}

// ---------- Sorted set operations ----------

func (p *prefixedPipeliner) ZAdd(ctx context.Context, key string, members ...baseRedis.Z) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "zadd")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZAdd(ctx, key, members...)
}

func (p *prefixedPipeliner) ZAddArgs(ctx context.Context, key string, args baseRedis.ZAddArgs) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "zadd")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZAddArgs(ctx, key, args)
}

func (p *prefixedPipeliner) ZAddArgsIncr(ctx context.Context, key string, args baseRedis.ZAddArgs) *baseRedis.FloatCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewFloatCmd(ctx, "zadd")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZAddArgsIncr(ctx, key, args)
}

func (p *prefixedPipeliner) ZCard(ctx context.Context, key string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "zcard")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZCard(ctx, key)
}

func (p *prefixedPipeliner) ZCount(ctx context.Context, key, minVal, maxVal string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "zcount")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZCount(ctx, key, minVal, maxVal)
}

func (p *prefixedPipeliner) ZIncrBy(ctx context.Context, key string, increment float64, member string) *baseRedis.FloatCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewFloatCmd(ctx, "zincrby")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZIncrBy(ctx, key, increment, member)
}

func (p *prefixedPipeliner) ZScore(ctx context.Context, key, member string) *baseRedis.FloatCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewFloatCmd(ctx, "zscore")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZScore(ctx, key, member)
}

func (p *prefixedPipeliner) ZMScore(ctx context.Context, key string, members ...string) *baseRedis.FloatSliceCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewFloatSliceCmd(ctx, "zmscore")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZMScore(ctx, key, members...)
}

func (p *prefixedPipeliner) ZRange(ctx context.Context, key string, start, stop int64) *baseRedis.StringSliceCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "zrange")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZRange(ctx, key, start, stop)
}

func (p *prefixedPipeliner) ZRangeArgs(ctx context.Context, z baseRedis.ZRangeArgs) *baseRedis.StringSliceCmd {
	key, err := p.client.prefixKey(z.Key)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "zrange")
		cmd.SetErr(err)

		return cmd
	}

	z.Key = key

	return p.Pipeliner.ZRangeArgs(ctx, z)
}

func (p *prefixedPipeliner) ZRangeArgsWithScores(ctx context.Context, z baseRedis.ZRangeArgs) *baseRedis.ZSliceCmd {
	key, err := p.client.prefixKey(z.Key)
	if err != nil {
		cmd := baseRedis.NewZSliceCmd(ctx, "zrange")
		cmd.SetErr(err)

		return cmd
	}

	z.Key = key

	return p.Pipeliner.ZRangeArgsWithScores(ctx, z)
}

func (p *prefixedPipeliner) ZRandMember(ctx context.Context, key string, count int) *baseRedis.StringSliceCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "zrandmember")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZRandMember(ctx, key, count)
}

func (p *prefixedPipeliner) ZRank(ctx context.Context, key, member string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "zrank")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZRank(ctx, key, member)
}

func (p *prefixedPipeliner) ZRem(ctx context.Context, key string, members ...any) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "zrem")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZRem(ctx, key, members...)
}

func (p *prefixedPipeliner) ZRevRange(ctx context.Context, key string, start, stop int64) *baseRedis.StringSliceCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewStringSliceCmd(ctx, "zrevrange")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZRevRange(ctx, key, start, stop)
}

func (p *prefixedPipeliner) ZRevRank(ctx context.Context, key, member string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "zrevrank")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.ZRevRank(ctx, key, member)
}
