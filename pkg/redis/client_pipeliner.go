package redis

import (
	"context"
	"time"

	baseRedis "github.com/go-redis/redis/v8"
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

func (p *prefixedPipeliner) MSet(ctx context.Context, pairs ...any) *baseRedis.StatusCmd {
	prefixedPairs, err := p.client.prefixInterleavedKeys(pairs...)
	if err != nil {
		cmd := baseRedis.NewStatusCmd(ctx, "mset")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.MSet(ctx, prefixedPairs...)
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

func (p *prefixedPipeliner) Incr(ctx context.Context, key string) *baseRedis.IntCmd {
	key, err := p.client.prefixKey(key)
	if err != nil {
		cmd := baseRedis.NewIntCmd(ctx, "incr")
		cmd.SetErr(err)

		return cmd
	}

	return p.Pipeliner.Incr(ctx, key)
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
