package dx

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
)

func runPostProcessorForDev(config cfg.GosoConf, postProcessor func(config cfg.GosoConf) error) (bool, error) {
	env, err := config.GetString("app.env", "")
	if err != nil {
		return false, err
	}

	if env != "dev" && env != "test" {
		return false, nil
	}

	if err := postProcessor(config); err != nil {
		return false, err
	}

	return true, nil
}
