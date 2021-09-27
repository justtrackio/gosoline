package dx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(1, "gosoline.dx.autoCreate", AutoCreateConfigPostProcessor)
}

func AutoCreateConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	return runPostProcessorForDev(config, func(config cfg.GosoConf) error {
		if err := config.Option(cfg.WithConfigSetting("dx.auto_create", true, cfg.SkipExisting)); err != nil {
			return fmt.Errorf("could not set dx.auto_create: %w", err)
		}

		return nil
	})
}

func ShouldAutoCreate(config cfg.Config) bool {
	return config.GetBool("dx.auto_create", false)
}
