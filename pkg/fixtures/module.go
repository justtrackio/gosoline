package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

type FixtureLoaderModule struct {
	kernel.BackgroundModule
	fixtureSets []*FixtureSet
}

func NewFixtureLoaderModule(fixtureSets []*FixtureSet) *FixtureLoaderModule {
	return &FixtureLoaderModule{
		fixtureSets: fixtureSets,
	}
}

func (m *FixtureLoaderModule) Boot(config cfg.Config, logger mon.Logger) error {
	loader := NewFixtureLoader(config, logger)
	return loader.Load(m.fixtureSets)
}

func (m *FixtureLoaderModule) Run(ctx context.Context) error {
	// do nothing: fixtures are loaded during boot
	return nil
}
