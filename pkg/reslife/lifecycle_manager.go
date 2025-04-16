package reslife

import (
	"context"
	"fmt"
	"regexp"

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
	//go:generate mockery --name Purger
	Purger interface {
		Purge(ctx context.Context) error
	}
	LifeCycleerFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (LifeCycleer, error)
)

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
	settings := ReadSettings(config)

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
	var err error
	var regexps []*regexp.Regexp
	var creator Creator

	if ok, regexps, err = m.prepareCycle(m.settings.Create); err != nil {
		return fmt.Errorf("can not prepare create cycle: %w", err)
	}

	if !ok {
		m.logger.Info("create lifecycle not enabled, skipping")

		return nil
	}

	if err := m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if m.shouldSkip(res.GetId(), m.created, regexps) {
			continue
		}

		now := m.clock.Now()
		if creator, ok = res.(Creator); !ok {
			continue
		}

		if err := creator.Create(ctx); err != nil {
			return fmt.Errorf("could not create resource %q: %w", res.GetId(), err)
		}

		took := m.clock.Since(now)
		m.logger.Info("created resource %s in %s", res.GetId(), took)
	}

	m.logger.Info("executed the create lifecycle")

	return nil
}

func (m *LifeCycleManager) Init(ctx context.Context) error {
	var ok bool
	var err error
	var regexps []*regexp.Regexp
	var initializer Initializer

	if ok, regexps, err = m.prepareCycle(m.settings.Init); err != nil {
		return fmt.Errorf("can not prepare init cycle: %w", err)
	}

	if !ok {
		m.logger.Info("init lifecycle not enabled, skipping")

		return nil
	}

	if err = m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if m.shouldSkip(res.GetId(), funk.Set[string]{}, regexps) {
			continue
		}

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
	var regexps []*regexp.Regexp
	var key string
	var data any
	var registerer Registerer

	if ok, regexps, err = m.prepareCycle(m.settings.Register); err != nil {
		return fmt.Errorf("can not prepare register cycle: %w", err)
	}

	if !ok {
		m.logger.Info("register lifecycle not enabled, skipping")

		return nil
	}

	if err = m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if m.shouldSkip(res.GetId(), m.registered, regexps) {
			continue
		}

		if registerer, ok = res.(Registerer); !ok {
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
	}

	m.logger.Info("executed the register lifecycle")

	return nil
}

func (m *LifeCycleManager) Purge(ctx context.Context) error {
	var ok bool
	var err error
	var regexps []*regexp.Regexp
	var purger Purger

	if ok, regexps, err = m.prepareCycle(m.settings.Purge); err != nil {
		return fmt.Errorf("can not prepare purge cycle: %w", err)
	}

	if !ok {
		m.logger.Info("purge lifecycle not enabled, skipping")

		return nil
	}

	if err := m.refreshResources(ctx); err != nil {
		return fmt.Errorf("could not refresh resources: %w", err)
	}

	for _, res := range m.resources {
		if m.shouldSkip(res.GetId(), m.purged, regexps) {
			continue
		}

		if purger, ok = res.(Purger); !ok {
			continue
		}

		now := m.clock.Now()
		if err := purger.Purge(ctx); err != nil {
			return fmt.Errorf("could not purge resource %q: %w", res.GetId(), err)
		}

		took := m.clock.Since(now)
		m.logger.Info("purged resource %s in %s", res.GetId(), took)
	}

	m.logger.Info("executed the purge lifecycle")

	return nil
}

func (m *LifeCycleManager) prepareCycle(settings SettingsCycle) (enabled bool, excludes []*regexp.Regexp, err error) {
	if !settings.Enabled {
		return false, nil, nil
	}

	regexps := make([]*regexp.Regexp, len(settings.Excludes))
	for i, pattern := range settings.Excludes {
		if regexps[i], err = regexp.Compile(pattern); err != nil {
			return false, nil, fmt.Errorf("could not compile exlude regexp %q: %w", pattern, err)
		}
	}

	return true, regexps, nil
}

func (m *LifeCycleManager) shouldSkip(id string, visited funk.Set[string], excludes []*regexp.Regexp) bool {
	if visited.Contains(id) {
		return true
	}

	visited.Set(id)

	for _, exclude := range excludes {
		if exclude.MatchString(id) {
			m.logger.Info("skipping resource %q as it is excluded", id)

			return true
		}
	}

	return false
}
