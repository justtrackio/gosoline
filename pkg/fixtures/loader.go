//go:build fixtures

package fixtures

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type fixtureLoader struct {
	logger   log.Logger
	settings *fixtureLoaderSettings
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger) FixtureLoader {
	settings := unmarshalFixtureLoaderSettings(config)

	return &fixtureLoader{
		logger:   logger.WithChannel("fixtures"),
		settings: settings,
	}
}

func (f *fixtureLoader) Load(ctx context.Context, group string, fixtureSets []FixtureSet) error {
	if !f.settings.Enabled {
		f.logger.Info("fixture loader is not enabled")

		return nil
	}

	f.logger.Info("loading fixtures")
	start := time.Now()
	defer func() {
		f.logger.Info("done loading fixtures in %s", time.Since(start))
	}()

	if !slices.Contains(f.settings.Groups, group) {
		f.logger.Info("fixture group %s is not enabled", group)

		return nil
	}

	for _, fixtureSet := range fixtureSets {
		f.logger.Info("loading fixtures for set %T", fixtureSet)

		if err := fixtureSet.Write(ctx); err != nil {
			return fmt.Errorf("failed to write fixtures: %w", err)
		}
	}

	return nil
}
