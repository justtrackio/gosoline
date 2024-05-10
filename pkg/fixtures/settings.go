package fixtures

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type fixtureLoaderSettings struct {
	Enabled bool     `cfg:"enabled" default:"false"`
	Groups  []string `cfg:"groups" default:"default"`
}

func unmarshalFixtureLoaderSettings(config cfg.Config) (*fixtureLoaderSettings, error) {
	settings := &fixtureLoaderSettings{}
	if err := config.UnmarshalKey("fixtures", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fixture loader settings: %w", err)
	}

	return settings, nil
}

func isFixtureLoadingEnabled(config cfg.Config) bool {
	return config.GetBool("fixtures.enabled", false)
}
