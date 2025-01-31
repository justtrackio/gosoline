package fixtures

import "github.com/justtrackio/gosoline/pkg/cfg"

type fixtureLoaderSettings struct {
	Enabled bool     `cfg:"enabled" default:"false"`
	Groups  []string `cfg:"groups" default:"default"`
}

func unmarshalFixtureLoaderSettings(config cfg.Config) *fixtureLoaderSettings {
	settings := &fixtureLoaderSettings{}
	config.UnmarshalKey("fixtures", settings)

	return settings
}

func isFixtureLoadingEnabled(config cfg.Config) bool {
	return config.GetBool("fixtures.enabled", false)
}
