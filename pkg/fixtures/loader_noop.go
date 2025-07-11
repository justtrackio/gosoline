//go:build !fixtures

package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type noopFixtureLoader struct {
	logger   log.Logger
	settings *fixtureLoaderSettings
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger, fixtureSets map[string][]FixtureSet, postProcessors []PostProcessor) (FixtureLoader, error) {
	settings, err := unmarshalFixtureLoaderSettings(config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal fixture loader settings: %w", err)
	}

	return &noopFixtureLoader{
		logger:   logger.WithChannel("fixture_loader"),
		settings: settings,
	}, nil
}

func (n *noopFixtureLoader) Load(ctx context.Context) error {
	if !n.settings.Enabled {
		return nil
	}

	n.logger.Info("fixtureSets loading disabled, to enable it use the 'fixtures' build tag")

	return nil
}
