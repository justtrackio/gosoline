package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

type FixtureSet struct {
	Enabled  bool
	Writer   FixtureWriterFactory
	Fixtures []interface{}
}

type FixtureLoader struct {
	kernel.BackgroundModule
	Writers     []FixtureWriter
	fixtureSets []*FixtureSet
}

func NewFixtureLoader(fixtureSets []*FixtureSet) *FixtureLoader {
	return &FixtureLoader{
		fixtureSets: fixtureSets,
	}
}

func (f *FixtureLoader) Boot(config cfg.Config, logger mon.Logger) error {
	logger = logger.WithChannel("fixture_loader")

	if !config.IsSet("fixture_loader_enabled") {
		logger.Info("fixture loader is not configured")
		return nil
	}

	if !config.GetBool("fixture_loader_enabled") {
		logger.Info("fixture loader is ot enabled")
		return nil
	}

	for _, fs := range f.fixtureSets {

		if !fs.Enabled {
			logger.Info("skipping disabled fixture set")
			continue
		}

		if fs.Writer == nil {
			return fmt.Errorf("fixture set is missing a writer")
		}

		writer := fs.Writer(config, logger)
		err := writer.WriteFixtures(fs)

		if err != nil {
			return fmt.Errorf("error during loading of fixture set: %w", err)
		}
	}

	return nil
}

func (f *FixtureLoader) Run(ctx context.Context) error {
	// do nothing: fixtures are loaded during boot
	return nil
}
