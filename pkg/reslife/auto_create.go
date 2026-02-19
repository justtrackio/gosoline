package reslife

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(1, "gosoline.reslife.autoCreate", AutoCreateConfigPostProcessor)
}

func AutoCreateConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	env, err := config.GetString("app.env", "")
	if err != nil {
		return false, fmt.Errorf("could not get app.env: %w", err)
	}

	if env != "dev" && env != "test" {
		return false, nil
	}

	if err := config.Option(cfg.WithConfigSetting("resource_lifecycles.create.enabled", true, cfg.SkipExisting)); err != nil {
		return false, fmt.Errorf("could not set reslife.create.enabled: %w", err)
	}

	return true, nil
}
