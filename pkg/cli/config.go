package cli

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

// UnmarshalFlags unmarshals CLI flag values from the application's cli.flags configuration section.
func UnmarshalFlags[T any](config cfg.Config) (*T, error) {
	var settings T
	var err error

	if settings, err = cfg.GetSettings[T](config, "cli.flags"); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flags: %w", err)
	}

	return &settings, nil
}

// GetArguments returns the positional arguments from the application's cli.args configuration section.
func GetArguments(config cfg.Config) ([]string, error) {
	return config.GetStringSlice("cli.args", []string{})
}
