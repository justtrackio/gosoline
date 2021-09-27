package dx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(1, "gosoline.dx.apiPort", ApiPortConfigPostProcessor)
}

func ApiPortConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	return runPostProcessorForDev(config, func(config cfg.GosoConf) error {
		if err := config.Option(cfg.WithConfigSetting("cfg.server.port", "0", cfg.SkipExisting)); err != nil {
			return fmt.Errorf("could not set cfg.server.port: %w", err)
		}

		if err := config.Option(cfg.WithConfigSetting("api.health.port", "0", cfg.SkipExisting)); err != nil {
			return fmt.Errorf("could not set api.health.port: %w", err)
		}

		return nil
	})
}
