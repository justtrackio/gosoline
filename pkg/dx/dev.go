package dx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func runPostProcessorForDev(config cfg.GosoConf, postProcessor func(config cfg.GosoConf) error) (bool, error) {
	env, err := config.GetString("env", "")
	if err != nil {
		return false, fmt.Errorf("failed to get env config: %w", err)
	}

	if env != "dev" && env != "test" {
		return false, nil
	}

	if err := postProcessor(config); err != nil {
		return false, err
	}

	return true, nil
}
