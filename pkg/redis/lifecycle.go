package redis

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKey = "redis.connections"

type Metadata struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	DB      int    `json:"db"`
}

type lifecycleManager struct {
	logger   log.Logger
	purger   *LifeCyclePurger
	settings *Settings
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Registerer
	reslife.Purger
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(settings *Settings) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var client Client

		if client, err = NewClientWithSettings(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("could not connect to database: %w", err)
		}

		return &lifecycleManager{
			logger:   logger,
			purger:   NewLifeCyclePurgerWithInterfaces(client),
			settings: settings,
		}, nil
	}
}

func (l lifecycleManager) GetId() string {
	return fmt.Sprintf("redis/%s", l.settings.Name)
}

func (l *lifecycleManager) Register(ctx context.Context) (key string, metadata any, err error) {
	metadata = Metadata{
		Name:    l.settings.Name,
		Address: l.settings.Address,
		DB:      l.settings.DB,
	}

	return MetadataKey, metadata, nil
}

func (l lifecycleManager) Purge(ctx context.Context) error {
	if err := l.purger.Purge(ctx); err != nil {
		return fmt.Errorf("can not purge: %w", err)
	}

	return nil
}
