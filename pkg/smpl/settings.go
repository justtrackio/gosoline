package smpl

import "github.com/justtrackio/gosoline/pkg/cfg"

// Settings configures the sampling behavior.
type Settings struct {
	// Enabled toggles the sampling logic. If false, sampling is assumed to be true but not stored in the context.
	Enabled bool `cfg:"enabled" default:"false"`
	// Strategies is a list of strategy names to apply in order.
	Strategies []string `cfg:"strategies"`
}

func UnmarshalSettings(config cfg.Config) (*Settings, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey("sampling", settings); err != nil {
		return nil, err
	}

	return settings, nil
}

func IsEnabled(config cfg.Config) (bool, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey("sampling", settings); err != nil {
		return false, err
	}

	return settings.Enabled, nil
}
