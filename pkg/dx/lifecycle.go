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
		Create(ctx context.Context) error
		Register(ctx context.Context) (string, any, error)
		Purge(ctx context.Context) error
	}
	LifeCycleerFactory    func(ctx context.Context, config cfg.Config, logger log.Logger) (LifeCycleer, error)
	lifeCyclePurgerCtxKey struct{}
)

func AddLifeCycleer(ctx context.Context, wr func() (string, LifeCycleerFactory)) error {
	var err error
	var manager map[string]LifeCycleerFactory

	if manager, err = provideLifeCycleers(ctx); err != nil {
		return fmt.Errorf("could not add life cycle purger: %w", err)
	}

	id, fc := wr()
	manager[id] = fc

	return nil
}

func provideLifeCycleers(ctx context.Context) (map[string]LifeCycleerFactory, error) {
	return appctx.Provide(ctx, lifeCyclePurgerCtxKey{}, func() (map[string]LifeCycleerFactory, error) {
		return make(map[string]LifeCycleerFactory), nil
	})
}

type LifeCycleManager struct {
	logger    log.Logger
	clock     clock.Clock
	resources map[string]LifeCycleer
}

func NewLifeCycleManager(ctx context.Context, config cfg.Config, logger log.Logger, clock clock.Clock) (*LifeCycleManager, error) {
	var err error
	var factories map[string]LifeCycleerFactory

	manager := &LifeCycleManager{
		logger:    logger,
		clock:     clock,
		resources: map[string]LifeCycleer{},
	}

	if factories, err = provideLifeCycleers(ctx); err != nil {
		return nil, fmt.Errorf("could not get lifecyleers: %w", err)
	}

	for id, fac := range factories {
		if manager.resources[id], err = fac(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("could not build lifecycleer with id %q: %w", id, err)
		}
	}

	return manager, nil
}

func (m *LifeCycleManager) Create(ctx context.Context) error {
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

func (m *LifeCycleManager) Register(ctx context.Context) error {
	var err error
	var key string
	var data any

	for id, res := range m.resources {
		if key, data, err = res.Register(ctx); err != nil {
			return fmt.Errorf("could not purge resource %q: %w", id, err)
		}

		if err = appctx.MetadataAppend(ctx, key, data); err != nil {
			return fmt.Errorf("can not access the appctx metadata: %w", err)
		}
	}

	return nil
}

func (m *LifeCycleManager) Purge(ctx context.Context) error {
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
	logger = logger.WithChannel("lifecycle-manager")

	var err error
	var manager *LifeCycleManager

	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func() {
			if manager, err = NewLifeCycleManager(ctx, config, logger, clock.Provider); err != nil {
				logger.Error("could not build lifecycle manager: %w", err)

				return
			}

			if err = manager.Create(ctx); err != nil {
				logger.Error("can not handle the create lifecycle: %w", err)

				return
			}

			if err = manager.Register(ctx); err != nil {
				logger.Error("can not handle the register lifecycle: %w", err)

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
