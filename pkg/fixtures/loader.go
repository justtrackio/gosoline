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

	for _, fixtureSet := range fixtureSets {
		if err := fixtureSet.Write(ctx); err != nil {
			return fmt.Errorf("failed to write fixtures: %w", err)
		}
	}

	return nil
}
