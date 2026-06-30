package cli

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func UnmarshalFlags[T any](config cfg.Config) (*T, error) {
	settings := new(T)

	if err := config.UnmarshalKey("cli.flags", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal flags: %w", err)
	}

	return settings, nil
}
