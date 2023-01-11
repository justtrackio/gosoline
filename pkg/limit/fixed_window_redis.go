package limit

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/redis"
)

func provideRedisClient(ctx context.Context, config cfg.Config, logger log.Logger) (redis.Client, error) {
	return appctx.Provide(ctx, limitPkgCtxKey("redis_client"), func() (redis.Client, error) {
		return redis.NewClient(config, logger, "rate_limits")
	})
}

func NewFixedWindowRedis(ctx context.Context, config cfg.Config, logger log.Logger, c FixedWindowConfig) (LimiterWithMiddleware, error) {
	redisClient, err := provideRedisClient(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	builder, err := newInvocationBuilder(c.Name)
	if err != nil {
		return nil, err
	}

	return NewFixedWindowRedisWithInterfaces(clock.NewRealClock(), redisClient, c, builder), nil
}

func NewFixedWindowRedisWithInterfaces(clock clock.Clock, redis redis.Client, config FixedWindowConfig, builder *invocationBuilder) LimiterWithMiddleware {
	backend := newFixedWindowRedisBackendWithInterfaces(redis, config.Window, config.Name)

	return NewFixedWindowLimiter(backend, clock, config, builder)
}

type fixedWindowRedis struct {
	redis  redis.Client
	window time.Duration
	name   string
}

func newFixedWindowRedisBackendWithInterfaces(redis redis.Client, window time.Duration, name string) *fixedWindowRedis {
	return &fixedWindowRedis{
		redis:  redis,
		window: window,
		name:   name,
	}
}

func (f fixedWindowRedis) Increment(ctx context.Context, prefix string) (incr *int, ttl *time.Duration, err error) {
	key := f.keyBuilder(prefix)

	pipe := f.redis.Pipeline().TxPipeline()
	increment := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, f.window)
	ttlCmd := pipe.TTL(ctx, key)
	if _, err = pipe.Exec(ctx); err != nil {
		return nil, nil, err
	}

	return mdl.Box(int(increment.Val())), mdl.Box(ttlCmd.Val()), nil
}

func (f fixedWindowRedis) keyBuilder(prefix string) string {
	return fmt.Sprintf("%s/%s", f.name, prefix)
}
