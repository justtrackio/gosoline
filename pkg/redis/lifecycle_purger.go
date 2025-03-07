package redis

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type LifeCyclePurger struct {
	client Client
}

func NewLifeCyclePurger(config cfg.Config, logger log.Logger, name string) (*LifeCyclePurger, error) {
	var err error
	var client Client

	settings := ReadSettings(config, name)

	if client, err = NewClientWithSettings(logger, settings); err != nil {
		return nil, fmt.Errorf("can not connect to database: %w", err)
	}

	return NewLifeCyclePurgerWithInterfaces(client), nil
}

func NewLifeCyclePurgerWithInterfaces(client Client) *LifeCyclePurger {
	return &LifeCyclePurger{
		client: client,
	}
}

func (p LifeCyclePurger) Purge(ctx context.Context) (err error) {
	if _, err = p.client.FlushDB(ctx); err != nil {
		return fmt.Errorf("can not flush database: %w", err)
	}

	return nil
}
