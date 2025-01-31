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
		GetId() string
	}
	Creator interface {
		Create(ctx context.Context) error
	}
	Initializer interface {
		Init(ctx context.Context) error
	}
	Registerer interface {
		Register(ctx context.Context) (string, any, error)
	}
	Purger interface {
		Purge(ctx context.Context) error
	}
	LifeCycleerFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (LifeCycleer, error)
)

type Settings struct {
	Create struct {
		Enabled bool `cfg:"enabled" default:"false"`
	} `cfg:"create"`
	Init struct {
		Enabled bool `cfg:"enabled" default:"true"`
	} `cfg:"init"`
	Register struct {
		Enabled bool `cfg:"enabled" default:"true"`
	} `cfg:"register"`
	Purge struct {
		Enabled bool `cfg:"enabled" default:"false"`
	} `cfg:"purge"`
}

type LifeCycleManager struct {
	logger     log.Logger
	clock      clock.Clock
	settings   *Settings
	refresher  func(fac LifeCycleerFactory) (LifeCycleer, error)
	resources  []LifeCycleer
	created    funk.Set[string]
	registered funk.Set[string]
	purged     funk.Set[string]
}

func NewLifeCycleManager(ctx context.Context, config cfg.Config, logger log.Logger) (*LifeCycleManager, error) {
	logger = logger.WithChannel("lifecycle-manager")

	settings := &Settings{}
	config.UnmarshalKey("resource_lifecycles", settings)

	return &LifeCycleManager{
		logger:   logger,
		clock:    clock.Provider,
		settings: settings,
		refresher: func(fac LifeCycleerFactory) (LifeCycleer, error) {
			return fac(ctx, config, logger)
		},
		resources:  nil,
		created:    funk.Set[string]{},
		registered: funk.Set[string]{},
		purged:     funk.Set[string]{},
	}, nil
}

func (m *LifeCycleManager) refreshResources(ctx context.Context) (err error) {
	var cont *container
	var resource LifeCycleer

	if cont, err = provideContainer(ctx); err != nil {
		err = fmt.Errorf("could not get lifecyleer factories: %w", err)

		return
	}

	cont.lck.Lock()
	defer cont.lck.Unlock()

	i := len(m.resources)
	for ; i < len(cont.factories); i++ {
		if resource, err = m.refresher(cont.factories[i]); err != nil {
			err = fmt.Errorf("could not build lifecycleer: %w", err)

			return
		}

		m.resources = append(m.resources, resource)
	}

	return nil
}

func (m *LifeCycleManager) Create(ctx context.Context) error {
	var ok bool
	var creator Creator

	if !m.settings.Create.Enabled {
		m.logger.Info("create lifecycle not enabled, skipping")

		return nil
	}

	if err := m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if m.created.Contains(res.GetId()) {
			continue
		}

		now := m.clock.Now()

		if creator, ok = res.(Creator); !ok {
			continue
		}

		if err := creator.Create(ctx); err != nil {
			return fmt.Errorf("could not create resource %q: %w", res.GetId(), err)
		}

		m.created.Set(res.GetId())

		took := m.clock.Since(now)
		m.logger.Info("created resource %s in %s", res.GetId(), took)
	}

	m.logger.Info("executed the create lifecycle")

	return nil
}

func (m *LifeCycleManager) Init(ctx context.Context) error {
	var ok bool
	var err error
	var initializer Initializer

	if !m.settings.Init.Enabled {
		m.logger.Info("init lifecycle not enabled, skipping")

		return nil
	}

	if err = m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if initializer, ok = res.(Initializer); !ok {
			continue
		}

		if err = initializer.Init(ctx); err != nil {
			return fmt.Errorf("could not init resource %q: %w", res.GetId(), err)
		}
	}

	m.logger.Info("executed the init lifecycle")

	return nil
}

func (m *LifeCycleManager) Register(ctx context.Context) error {
	var ok bool
	var err error
	var key string
	var data any
	var registerer Registerer

	if !m.settings.Register.Enabled {
		m.logger.Info("register lifecycle not enabled, skipping")

		return nil
	}

	if err = m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if registerer, ok = res.(Registerer); !ok {
			continue
		}

		if m.registered.Contains(res.GetId()) {
			continue
		}

		if key, data, err = registerer.Register(ctx); err != nil {
			return fmt.Errorf("could not register resource %q: %w", res.GetId(), err)
		}

		if key == "" || data == nil {
			continue
		}

		if err = appctx.MetadataAppend(ctx, key, data); err != nil {
			return fmt.Errorf("can not access the appctx metadata: %w", err)
		}

		m.registered.Set(res.GetId())
	}

	m.logger.Info("executed the register lifecycle")

	return nil
}

func (m *LifeCycleManager) Purge(ctx context.Context) error {
	var ok bool
	var purger Purger

	if !m.settings.Purge.Enabled {
		m.logger.Info("purge lifecycle not enabled, skipping")

		return nil
	}

	if err := m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if purger, ok = res.(Purger); !ok {
			continue
		}

		if m.purged.Contains(res.GetId()) {
			continue
		}

		now := m.clock.Now()

		if err := purger.Purge(ctx); err != nil {
			return fmt.Errorf("could not purge resource %q: %w", res.GetId(), err)
		}

		m.purged.Set(res.GetId())

		took := m.clock.Since(now)
		m.logger.Info("purged resource %s in %s", res.GetId(), took)
	}

	m.logger.Info("executed the purge lifecycle")

	return nil
}
