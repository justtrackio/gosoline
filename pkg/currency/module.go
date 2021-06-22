package currency

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"time"
)

type Module struct {
	kernel.BackgroundModule
	kernel.ServiceStage
	updaterService UpdaterService
	logger         log.Logger
}

func NewCurrencyModule() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		updater, err := NewUpdater(config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create updater: %w", err)
		}

		module := &Module{
			logger:         logger,
			updaterService: updater,
		}

		return module, nil
	}
}

func (module *Module) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(1) * time.Hour)
	module.refresh(ctx)
	module.importExchangeRates(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			module.refresh(ctx)
		}
	}
}

func (module *Module) refresh(ctx context.Context) {
	err := module.updaterService.EnsureRecentExchangeRates(ctx)
	if err != nil {
		module.logger.Error("failed to refresh currency exchange rates: %w", err)
	}
}

func (module *Module) importExchangeRates(ctx context.Context) {
	err := module.updaterService.EnsureHistoricalExchangeRates(ctx)
	if err != nil {
		module.logger.Error("failed to import historical currency exchange rates: %w", err)
	}
}
