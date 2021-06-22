package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/redis"
)

type redisPurger struct {
	logger log.Logger
	client redis.Client
	name   *string
}

func newRedisPurger(config cfg.Config, logger log.Logger, name *string) (*redisPurger, error) {
	client, err := redis.ProvideClient(config, logger, *name)
	if err != nil {
		return nil, fmt.Errorf("can not create redis client: %w", err)
	}

	return &redisPurger{
		logger: logger,
		name:   name,
		client: client,
	}, nil
}

func (p *redisPurger) purge() error {
	p.logger.Info("flushing redis %s", *p.name)
	_, err := p.client.FlushDB(context.Background())
	p.logger.Info("flushing redis %s done", *p.name)

	return err
}
