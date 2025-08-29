package currency

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	lockResourceRecentExchangeRates     = "recentExchangeRates"
	lockResourceHistoricalExchangeRates = "historicalExchangeRates"
)

type Module struct {
	kernel.EssentialBackgroundModule
	kernel.ServiceStage
	logger         log.Logger
	updaterService UpdaterService
	healthy        *atomic.Bool
	lockProvider   conc.DistributedLockProvider
}

// ensure interface compatibility
var _ kernel.HealthCheckedModule = Module{}

func NewCurrencyModule() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		updater, err := NewUpdater(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create updater: %w", err)
		}

		appId, err := cfg.GetAppIdFromConfig(config)
		if err != nil {
			return nil, fmt.Errorf("can not get app id from config: %w", err)
		}

		lockProvider, err := ddb.NewDdbLockProvider(ctx, config, logger, conc.DistributedLockSettings{
			AppId:           appId,
			DefaultLockTime: 3 * time.Minute,
			Domain:          "currency",
		})
		if err != nil {
			return nil, fmt.Errorf("can not create lock provider: %w", err)
		}

		module := Module{
			logger:         logger,
			updaterService: updater,
			healthy:        &atomic.Bool{},
			lockProvider:   lockProvider,
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
	if err := module.updateExchangeRates(ctx, lockResourceRecentExchangeRates, module.updaterService.EnsureRecentExchangeRates); err != nil {
		if exec.IsRequestCanceled(err) {
			return nil
		}

		return fmt.Errorf("failed to fetch initial recent exchange rates: %w", err)
	}

	if err := module.updateExchangeRates(ctx, lockResourceHistoricalExchangeRates, module.updaterService.EnsureHistoricalExchangeRates); err != nil {
		if exec.IsRequestCanceled(err) {
			return nil
		}

		return fmt.Errorf("failed to fetch initial historical exchange rates: %w", err)
	}

	module.healthy.Store(true)

	ticker := time.NewTicker(1 * time.Hour)
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			if err := module.updateExchangeRates(ctx, lockResourceRecentExchangeRates, module.updaterService.EnsureRecentExchangeRates); err != nil {
				if exec.IsRequestCanceled(err) {
					return nil
				}

				module.logger.Error(ctx, "failed to refresh recent exchange rates: %w", err)
			}
		}
	}
}

func (module Module) updateExchangeRates(ctx context.Context, lockResource string, updateFunc func(context.Context) error) error {
	lock, err := module.lockProvider.Acquire(ctx, lockResource)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for resource %s: %w", lockResource, err)
	}

	errs := &multierror.Error{}

	if err := updateFunc(ctx); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to run update for %s: %w", lockResource, err))
	}

	if err := lock.Release(); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to release lock for %s: %w", lockResource, err))
	}

	return errs.ErrorOrNil()
}
