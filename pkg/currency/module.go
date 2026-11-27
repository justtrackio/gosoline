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

	currencyLockTime          = 3 * time.Minute
	currencyLockRenewInterval = currencyLockTime / 3
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

// NewCurrencyModule returns a kernel module factory for the currency module.
// The module will start a background process that periodically updates exchange rates from configured providers.
// During a refresh the module renews its lease and cancels the update if renewal fails.
// By default, it will update historical exchange rates for the last 30 days.
// If you need to create a new table with all the data, it is recommended to copy data from an existing table.
func NewCurrencyModule() kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		updater, err := NewUpdater(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create updater: %w", err)
		}

		identity, err := cfg.GetAppIdentity(config)
		if err != nil {
			return nil, fmt.Errorf("can not get app identity from config: %w", err)
		}

		lockProvider, err := ddb.NewDdbLockProvider(ctx, config, logger, conc.DistributedLockSettings{
			Identity:        identity,
			DefaultLockTime: currencyLockTime,
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
		if ctx.Err() != nil && exec.IsRequestCanceled(err) {
			return nil
		}

		return fmt.Errorf("failed to fetch initial recent exchange rates: %w", err)
	}

	if err := module.updateExchangeRates(ctx, lockResourceHistoricalExchangeRates, module.updaterService.EnsureHistoricalExchangeRates); err != nil {
		if ctx.Err() != nil && exec.IsRequestCanceled(err) {
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
				if ctx.Err() != nil && exec.IsRequestCanceled(err) {
					return nil
				}

				module.logger.Error(ctx, "failed to refresh recent exchange rates: %w", err)
			}
		}
	}
}

func (module Module) updateExchangeRates(ctx context.Context, lockResource string, updateFunc func(context.Context) error) error {
	leaseRequested := time.Now()
	lock, err := module.lockProvider.Acquire(ctx, lockResource)
	if err != nil {
		return fmt.Errorf("failed to acquire lock for resource %s: %w", lockResource, err)
	}

	updateErr, renewErr := runCurrencyRefresh(ctx, lock, leaseRequested.Add(currencyLockRenewInterval), updateFunc)

	errs := &multierror.Error{}

	if renewErr != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to renew lock for %s: %w", lockResource, renewErr))
	}

	// A canceled update is expected after renewal fails; report the cause instead.
	if updateErr != nil && (renewErr == nil || !exec.IsRequestCanceled(updateErr)) {
		errs = multierror.Append(errs, fmt.Errorf("failed to run update for %s: %w", lockResource, updateErr))
	}

	if err := lock.Release(); err != nil {
		errs = multierror.Append(errs, fmt.Errorf("failed to release lock for %s: %w", lockResource, err))
	}

	return errs.ErrorOrNil()
}

func runCurrencyRefresh(ctx context.Context, lock conc.DistributedLock, renewAt time.Time, updateFunc func(context.Context) error) (updateErr, renewErr error) {
	if now := time.Now(); !now.Before(renewAt) {
		// Acquire may have waited on DynamoDB while the lease was already aging.
		// Confirm ownership before starting a refresh if we missed the first renewal time.
		renewAt, renewErr = renewCurrencyLock(ctx, lock, now.Add(currencyLockRenewInterval))
		if renewErr != nil {
			return nil, renewErr
		}
	}

	updateCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	finished := make(chan struct{})
	result := make(chan error)
	go func() {
		err := maintainCurrencyLock(ctx, lock, renewAt, finished)
		if err != nil {
			cancel()
		}
		result <- err
	}()
	defer func() {
		close(finished)
		// Let an in-flight renewal finish before releasing the lock.
		renewErr = <-result
	}()

	// Stay on the module goroutine so the kernel can recover provider panics.
	return updateFunc(updateCtx), nil
}

func maintainCurrencyLock(ctx context.Context, lock conc.DistributedLock, renewAt time.Time, finished <-chan struct{}) error {
	timer := time.NewTimer(time.Until(renewAt))
	defer timer.Stop()

	for {
		select {
		case <-finished:
			return nil
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

		// Leave time for the refresh to stop before the current lease expires.
		nextRenewal, err := renewCurrencyLock(ctx, lock, renewAt.Add(currencyLockRenewInterval))
		if err != nil {
			return err
		}
		renewAt = nextRenewal
		timer.Reset(time.Until(renewAt))
	}
}

func renewCurrencyLock(ctx context.Context, lock conc.DistributedLock, deadline time.Time) (time.Time, error) {
	// DynamoDB starts counting the new lease before the request returns.
	nextRenewal := time.Now().Add(currencyLockRenewInterval)
	renewCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	if err := renewCtx.Err(); err != nil {
		return time.Time{}, err
	}
	if err := lock.Renew(renewCtx, currencyLockTime); err != nil {
		return time.Time{}, err
	}

	return nextRenewal, renewCtx.Err()
}
