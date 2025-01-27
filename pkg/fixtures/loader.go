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
	postProcessors []PostProcessor
	settings       *fixtureLoaderSettings
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger, postProcessorFactories ...PostProcessorFactory) (FixtureLoader, error) {
	var err error

	settings := unmarshalFixtureLoaderSettings(config)
	postProcessors := make([]PostProcessor, len(postProcessorFactories))

	for i, postProcessorFactory := range postProcessorFactories {
		if postProcessors[i], err = postProcessorFactory(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not build fixture post processor #%d: %w", i, err)
		}
	}

	return &fixtureLoader{
		logger:         logger.WithChannel("fixtures"),
		postProcessors: postProcessors,
		settings:       settings,
	}, nil
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

	for _, processor := range f.postProcessors {
		if err := processor.Process(ctx); err != nil {
			return fmt.Errorf("can not post process fixtures: %w", err)
		}
	}

	return nil
}
