package dx

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	LifeCycleer interface {
		GetId() string
		Create(ctx context.Context) error
		Purge(ctx context.Context) error
	}
	lifeCyclePurgerCtxKey struct{}
)

func AddLifeCycleer(ctx context.Context, logger log.Logger, lc LifeCycleer) error {
	var err error
	var manager *lifeCycleManager

	if manager, err = provideLifeCycleManager(ctx, logger); err != nil {
		return fmt.Errorf("could not add life cycle purger: %w", err)
	}

	manager.resources[lc.GetId()] = lc

	return nil
}

func provideLifeCycleManager(ctx context.Context, logger log.Logger) (*lifeCycleManager, error) {
	return appctx.Provide(ctx, lifeCyclePurgerCtxKey{}, func() (*lifeCycleManager, error) {
		return &lifeCycleManager{
			logger:    logger.WithChannel("lifecycle"),
			clock:     clock.Provider,
			resources: map[string]LifeCycleer{},
		}, nil
	})
}

type lifeCycleManager struct {
	logger    log.Logger
	clock     clock.Clock
	resources map[string]LifeCycleer
}

func (m *lifeCycleManager) Create(ctx context.Context) error {
	for id, res := range m.resources {
		now := m.clock.Now()

		if err := res.Create(ctx); err != nil {
			return fmt.Errorf("could not purge resource %q: %w", id, err)
		}

		took := m.clock.Since(now)
		m.logger.Info("created resource %s in %s", id, took)
	}

	return nil
}

func (m *lifeCycleManager) Purge(ctx context.Context) error {
	for id, res := range m.resources {
		now := m.clock.Now()

		if err := res.Purge(ctx); err != nil {
			return fmt.Errorf("could not purge resource %q: %w", id, err)
		}

		took := m.clock.Since(now)
		m.logger.Info("purged resource %s in %s", id, took)
	}

	return nil
}

func KernelLifeCycleManager(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	var err error
	var manager *lifeCycleManager

	if manager, err = provideLifeCycleManager(ctx, logger); err != nil {
		return nil, fmt.Errorf("could not add life cycle purger: %w", err)
	}

	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func() {
			if err = manager.Create(ctx); err != nil {
				logger.Error("can not handle the create lifecycle: %w", err)

				return
			}

			if err = manager.Purge(ctx); err != nil {
				logger.Error("can not handle the purge lifecycle: %w", err)

				return
			}

			next()
		}
	}, nil
}
