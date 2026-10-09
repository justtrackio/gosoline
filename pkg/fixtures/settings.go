package fixtures

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type fixtureLoaderSettings struct {
	Enabled bool     `cfg:"enabled" default:"false"`
	Groups  []string `cfg:"groups" default:"default"`
}

func unmarshalFixtureLoaderSettings(config cfg.Config) (fixtureLoaderSettings, error) {
	var settings fixtureLoaderSettings
	var err error

	if settings, err = cfg.GetSettings[fixtureLoaderSettings](config, "fixtures"); err != nil {
		return fixtureLoaderSettings{}, fmt.Errorf("failed to unmarshal fixture loader settings: %w", err)
	}

	return settings, nil
}

func isFixtureLoadingEnabled(config cfg.Config) (bool, error) {
	return config.GetBool("fixtures.enabled", false)
}
