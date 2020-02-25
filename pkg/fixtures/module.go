package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

type FixtureLoaderModule struct {
	kernel.BackgroundModule
	loader *FixtureLoader
}

func NewFixtureLoaderModule(fixtureSets []*FixtureSet) *FixtureLoaderModule {
	loader := &FixtureLoader{
		fixtureSets: fixtureSets,
	}

	return &FixtureLoaderModule{
		loader: loader,
	}
}

func (m *FixtureLoaderModule) Boot(config cfg.Config, logger mon.Logger) error {
	return m.loader.Load(config, logger)
}

func (m *FixtureLoaderModule) Run(ctx context.Context) error {
	// do nothing: fixtures are loaded during boot
	return nil
}
