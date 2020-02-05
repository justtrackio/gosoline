package currency

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

type Module struct {
	kernel.BackgroundModule
	updaterService UpdaterService
	logger         mon.Logger
}

func (module *Module) Boot(config cfg.Config, logger mon.Logger) error {
	module.updaterService = NewUpdater(config, logger)
	module.logger = logger

	return nil
}

func (module *Module) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(1) * time.Hour)
	module.refresh(ctx)
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
		module.logger.Error(err, "failed to refresh currency exchange rates")
	}
}

func NewCurrencyModule() *Module {
	return &Module{}
}
