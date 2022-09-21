package currency

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Module struct {
	kernel.BackgroundModule
	kernel.ServiceStage
	updater UpdaterService
	logger  log.Logger
}

func NewCurrencyModule() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		updater, err := NewUpdater(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create updater: %w", err)
		}

		return NewCurrencyModuleWithInterfaces(logger, updater), nil
	}
}

func NewCurrencyModuleWithInterfaces(logger log.Logger, updater UpdaterService) kernel.Module {
	return &Module{
		logger:  logger,
		updater: updater,
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
	err := module.updater.EnsureRecentExchangeRates(ctx)
	if err != nil {
		module.logger.Error("failed to refresh currency exchange rates: %w", err)
	}
}

func (module *Module) importExchangeRates(ctx context.Context) {
	err := module.updater.EnsureHistoricalExchangeRates(ctx)
	if err != nil {
		module.logger.Error("failed to import historical currency exchange rates: %w", err)
	}
}
