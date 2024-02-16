package ipread

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type RefreshModule struct {
	kernel.BackgroundModule
	kernel.ServiceStage
	kernel.HealthCheckedModule
	logger   log.Logger
	provider Provider
	healthy  *atomic.Bool
	settings RefreshSettings
}

func RefreshModuleFactory(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
	modules := map[string]kernel.ModuleFactory{}
	readerSettings := readAllSettings(config)

	for name, settings := range readerSettings {
		moduleName := fmt.Sprintf("ipread-refresh-%s", name)
		modules[moduleName] = NewProviderRefreshModule(name, settings.Refresh)
	}

	return modules, nil
}

func NewProviderRefreshModule(name string, settings RefreshSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		logger = logger.WithChannel("ipread")

		var err error
		var read *reader

		if read, err = ProvideReader(ctx, config, logger, name); err != nil {
			return nil, fmt.Errorf("can not get reader with name %s: %w", name, err)
		}

		module := &RefreshModule{
			logger:   logger,
			provider: read.provider,
			healthy:  &atomic.Bool{},
			settings: settings,
		}

		return module, nil
	}
}

func (m *RefreshModule) IsHealthy(ctx context.Context) (bool, error) {
	return m.healthy.Load(), nil
}

func (m *RefreshModule) Run(ctx context.Context) (err error) {
	defer func() {
		m.healthy.Store(false)

		if closeErr := m.provider.Close(); closeErr != nil {
			err = multierror.Append(err, fmt.Errorf("can not close ipread provider: %w", closeErr))
		}

		m.provider = nil
	}()

	if err = m.provider.Refresh(ctx); err != nil {
		return fmt.Errorf("can not refresh provider: %w", err)
	}

	m.healthy.Store(true)

	if !m.settings.Enabled {
		return nil
	}

	ticker := time.NewTicker(m.settings.Interval)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err = m.provider.Refresh(ctx); err != nil {
				m.logger.Error("can not refresh provider: %w", err)
			}
		}
	}
}
