package fixtures

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
)

type redisPurger struct {
	logger mon.Logger
	client redis.Client
	name   *string
}

func newRedisPurger(config cfg.Config, logger mon.Logger, name *string) (*redisPurger, error) {
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
	p.logger.Infof("flushing redis %s", *p.name)
	_, err := p.client.FlushDB()
	p.logger.Infof("flushing redis %s done", *p.name)

	return err
}
