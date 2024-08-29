package currency

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Module struct {
	kernel.EssentialBackgroundModule
	kernel.ServiceStage
	logger         log.Logger
	updaterService UpdaterService
	healthy        *atomic.Bool
}

// ensure interface compatibility
var _ kernel.HealthCheckedModule = Module{}

func NewCurrencyModule() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		updater, err := NewUpdater(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create updater: %w", err)
		}

		module := Module{
			logger:         logger,
			updaterService: updater,
			healthy:        &atomic.Bool{},
		}

		return module, nil
	}
}

func (module Module) IsHealthy(ctx context.Context) (bool, error) {
	return module.healthy.Load(), nil
}

func (module Module) Run(ctx context.Context) error {
	defer module.healthy.Store(false)

	// load historical and current data, then the module is healthy
	if err := module.updaterService.EnsureRecentExchangeRates(ctx); err != nil {
		return fmt.Errorf("failed to fetch initial rates: %w", err)
	}
	if err := module.updaterService.EnsureHistoricalExchangeRates(ctx); err != nil {
		return fmt.Errorf("failed to fetch initial historical rates: %w", err)
	}

	module.healthy.Store(true)

	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := module.updaterService.EnsureRecentExchangeRates(ctx); err != nil {
				// we already have some data, let's try again in an hour
				module.logger.Error("failed to refresh currency exchange rates: %w", err)
			}
		}
	}
}
