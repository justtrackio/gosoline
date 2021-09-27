package dx

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func runPostProcessorForDev(config cfg.GosoConf, postProcessor func(config cfg.GosoConf) error) (bool, error) {
	env := config.GetString("env", "")

	if env != "dev" && env != "test" {
		return false, nil
	}

	if err := postProcessor(config); err != nil {
		return false, err
	}

	return true, nil
}
