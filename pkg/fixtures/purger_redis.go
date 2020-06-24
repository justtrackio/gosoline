package fixtures

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
)

type redisPurger struct {
	logger mon.Logger
	client redis.Client
	name   *string
}

func newRedisPurger(config cfg.Config, logger mon.Logger, name *string) *redisPurger {
	client := redis.ProvideClient(config, logger, *name)

	return &redisPurger{
		logger: logger,
		name:   name,
		client: client,
	}
}

func (p *redisPurger) purge() error {
	p.logger.Infof("flushing redis %s", *p.name)
	_, err := p.client.FlushDB()
	p.logger.Infof("flushing redis %s done", *p.name)

	return err
}
