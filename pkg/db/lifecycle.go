package db

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

const MetadataKey = "db.connections"

type Metadata struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

type lifecycleManager struct {
	logger   log.Logger
	name     string
	settings *Settings
	purger   reslife.Purger
}

type LifecycleManager interface {
	reslife.LifeCycleer
	reslife.Registerer
	reslife.Purger
}

var _ LifecycleManager = (*lifecycleManager)(nil)

func NewLifecycleManager(name string, settings *Settings) reslife.LifeCycleerFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
		var err error
		var purger reslife.Purger

		if purger, err = NewLifeCyclePurgerWithSettings(logger, settings); err != nil {
			return nil, err
		}

		return &lifecycleManager{
			logger:   logger,
			name:     name,
			settings: settings,
			purger:   purger,
		}, nil
	}
}

func (m *lifecycleManager) GetId() string {
	return fmt.Sprintf("db/%s", m.settings.Uri.Database)
}

func (m *lifecycleManager) Register(_ context.Context) (key string, metadata any, err error) {
	metadata = Metadata{
		Name:     m.name,
		Host:     m.settings.Uri.Host,
		Port:     m.settings.Uri.Port,
		Database: m.settings.Uri.Database,
	}

	return MetadataKey, metadata, nil
}

func (m *lifecycleManager) Purge(ctx context.Context) (err error) {
	return m.purger.Purge(ctx)
}
