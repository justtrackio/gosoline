//go:build fixtures

package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type fixtureLoaderSettings struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

type fixtureLoader struct {
	logger   log.Logger
	settings *fixtureLoaderSettings
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger) FixtureLoader {
	logger = logger.WithChannel("fixture_loader")

	settings := &fixtureLoaderSettings{}
	config.UnmarshalKey("fixtures", settings)

	return &fixtureLoader{
		logger:   logger,
		settings: settings,
	}
}

func (f *fixtureLoader) Load(ctx context.Context, fixtureSets []FixtureSet) error {
	if !f.settings.Enabled {
		f.logger.Info("fixture loader is not enabled")
		return nil
	}

	for _, fs := range fixtureSets {
		if err := fs.Write(ctx); err != nil {
			return fmt.Errorf("failed to load fixtures: %w", err)
		}
	}

	return nil
}
