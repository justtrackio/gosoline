package redis

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

type lifecycleManager struct {
	logger log.Logger
	client Client
	name   string
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Purger
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(name string) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var client Client

		if client, err = ProvideClient(ctx, config, logger, name); err != nil {
			return nil, fmt.Errorf("could not connect to database: %w", err)
		}

		return &lifecycleManager{
			logger: logger,
			client: client,
			name:   name,
		}, nil
	}
}

func (l lifecycleManager) GetId() string {
	return fmt.Sprintf("redis/%s", l.name)
}

func (l lifecycleManager) Purge(ctx context.Context) error {
	if _, err := l.client.FlushDB(ctx); err != nil {
		return fmt.Errorf("could not flush database: %w", err)
	}

	return nil
}
