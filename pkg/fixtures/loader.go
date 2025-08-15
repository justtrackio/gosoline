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
	logger         log.Logger
	fixtureSets    map[string][]FixtureSet
	postProcessors []PostProcessor
	settings       *fixtureLoaderSettings
}

func NewFixtureLoader(_ context.Context, config cfg.Config, logger log.Logger, fixtureSets map[string][]FixtureSet, postProcessors []PostProcessor) (FixtureLoader, error) {
	settings, err := unmarshalFixtureLoaderSettings(config)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal fixture loader settings: %w", err)
	}

	return &fixtureLoader{
		logger:         logger.WithChannel("fixtures"),
		fixtureSets:    fixtureSets,
		postProcessors: postProcessors,
		settings:       settings,
	}, nil
}

func (f *fixtureLoader) Load(ctx context.Context) error {
	logger := f.logger.WithContext(ctx)

	if !f.settings.Enabled {
		logger.Info("fixture loader is not enabled")

		return nil
	}

	logger.Info("loading fixtures")
	start := time.Now()
	defer func() {
		logger.Info("done loading fixtures in %s", time.Since(start))
	}()

	for group, fixtureSets := range f.fixtureSets {
		if !slices.Contains(f.settings.Groups, group) {
			logger.Info("fixture group %s is not enabled", group)

			continue
		}

		for _, fixtureSet := range fixtureSets {
			logger.Info("loading fixtures for set %T", fixtureSet)

			if err := fixtureSet.Write(ctx); err != nil {
				return fmt.Errorf("failed to write fixtures: %w", err)
			}
		}
	}

	for _, processor := range f.postProcessors {
		if err := processor.Process(ctx); err != nil {
			return fmt.Errorf("can not post process fixtures: %w", err)
		}
	}

	return nil
}
