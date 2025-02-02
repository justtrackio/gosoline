package reslife

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	LifeCycleer interface {
		Create(ctx context.Context) error
		Register(ctx context.Context) (string, any, error)
		Purge(ctx context.Context) error
	}
	LifeCycleerFactory       func(ctx context.Context, config cfg.Config, logger log.Logger) (LifeCycleer, error)
	LifeCycleerFactoryCtxKey struct{}
	lifeCycleerCtxKey        struct{}
)

type lifeCycleerFactoryContainer struct {
	factories []LifeCycleerFactory
	ids       []string
}

type lifeCycleerContainer struct {
	resources []LifeCycleer
	ids       []string
}

func AddLifeCycleer(ctx context.Context, wr func() (string, LifeCycleerFactory)) error {
	var err error
	var container *lifeCycleerFactoryContainer

	if container, err = provideLifeCycleers(ctx); err != nil {
		return fmt.Errorf("could not add life cycle purger: %w", err)
	}

	id, fc := wr()

	if funk.Contains(container.ids, id) {
		return nil
	}

	container.factories = append(container.factories, fc)
	container.ids = append(container.ids, id)

	return nil
}

func provideLifeCycleers(ctx context.Context) (*lifeCycleerFactoryContainer, error) {
	return appctx.Provide(ctx, lifeCycleerCtxKey{}, func() (*lifeCycleerFactoryContainer, error) {
		return &lifeCycleerFactoryContainer{
			factories: make([]LifeCycleerFactory, 0),
			ids:       make([]string, 0),
		}, nil
	})
}

type LifeCycleManager struct {
	logger    log.Logger
	clock     clock.Clock
	refresher func(fac LifeCycleerFactory) (LifeCycleer, error)
	container *lifeCycleerContainer
	created   []string
}

func ProvideLifeCycleManager(ctx context.Context, config cfg.Config, logger log.Logger) (*LifeCycleManager, error) {
	return appctx.Provide(ctx, LifeCycleerFactoryCtxKey{}, func() (*LifeCycleManager, error) {
		return &LifeCycleManager{
			logger: logger,
			clock:  clock.Provider,
			refresher: func(fac LifeCycleerFactory) (LifeCycleer, error) {
				return fac(ctx, config, logger)
			},
			container: &lifeCycleerContainer{
				resources: nil,
				ids:       nil,
			},
			created: []string{},
		}, nil
	})
}

func (m *LifeCycleManager) refreshResources(ctx context.Context) error {
	var err error
	var container *lifeCycleerFactoryContainer
	var resource LifeCycleer

	if container, err = provideLifeCycleers(ctx); err != nil {
		return fmt.Errorf("could not get lifecyleers: %w", err)
	}

	for i, id := range container.ids {
		if funk.Contains(m.container.ids, id) {
			continue
		}

		if resource, err = m.refresher(container.factories[i]); err != nil {
			return fmt.Errorf("could not build lifecycleer with id %q: %w", id, err)
		}

		m.container.resources = append(m.container.resources, resource)
		m.container.ids = append(m.container.ids, id)
	}

	return nil
}

func (m *LifeCycleManager) Create(ctx context.Context) error {
	if err := m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for i, res := range m.container.resources {
		if funk.Contains(m.created, m.container.ids[i]) {
			continue
		}

		now := m.clock.Now()

		if err := res.Create(ctx); err != nil {
			return fmt.Errorf("could not purge resource %q: %w", m.container.ids[i], err)
		}

		m.created = append(m.created, m.container.ids[i])

		took := m.clock.Since(now)
		m.logger.Info("created resource %s in %s", m.container.ids[i], took)
	}

	return nil
}

func (m *LifeCycleManager) Register(ctx context.Context) error {
	var err error
	var key string
	var data any

	if err = m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for i, res := range m.container.resources {
		if key, data, err = res.Register(ctx); err != nil {
			return fmt.Errorf("could not register resource %q: %w", m.container.ids[i], err)
		}

		if err = appctx.MetadataAppend(ctx, key, data); err != nil {
			return fmt.Errorf("can not access the appctx metadata: %w", err)
		}
	}

	return nil
}

func (m *LifeCycleManager) Purge(ctx context.Context) error {
	if err := m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for i, res := range m.container.resources {
		now := m.clock.Now()

		if err := res.Purge(ctx); err != nil {
			return fmt.Errorf("could not purge resource %q: %w", m.container.ids[i], err)
		}

		took := m.clock.Since(now)
		m.logger.Info("purged resource %s in %s", m.container.ids[i], took)
	}

	return nil
}
