//go:build !fixtures

package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type noopFixtureLoader struct {
	logger   log.Logger
	settings *fixtureLoaderSettings
}

func NewFixtureLoader(ctx context.Context, config cfg.Config, logger log.Logger, postProcessorFactories ...PostProcessorFactory) (FixtureLoader, error) {
	settings := unmarshalFixtureLoaderSettings(config)

	return &noopFixtureLoader{
		logger:   logger.WithChannel("fixture_loader"),
		settings: settings,
	}, nil
}

func (n *noopFixtureLoader) Load(ctx context.Context, group string, fixtureSets []FixtureSet) error {
	if !n.settings.Enabled {
		return nil
	}

	n.logger.Info("fixtureSets loading disabled, to enable it use the 'fixtures' build tag")

	return nil
}
