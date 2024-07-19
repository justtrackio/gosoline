package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/redis"
)

type redisPurger struct {
	logger log.Logger
	client redis.Client
	name   string
}

func NewRedisPurger(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*redisPurger, error) {
	client, err := redis.ProvideClient(ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	return &redisPurger{
		logger: logger.WithChannel("redis-purger"),
		name:   name,
		client: client,
	}, nil
}

func (p *redisPurger) Purge(ctx context.Context) error {
	p.logger.Info("flushing redis %s", p.name)
	_, err := p.client.FlushDB(ctx)
	p.logger.Info("flushing redis %s done", p.name)

	return err
}
