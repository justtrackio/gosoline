package dx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(1, "gosoline.dx.useRandomPort", UseRandomPortConfigPostProcessor)
}

var randomizablePortSettings = make(map[string]struct{})

func RegisterRandomizablePortSetting(setting string) {
	randomizablePortSettings[setting] = struct{}{}
}

func UseRandomPortConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	return runPostProcessorForDev(config, func(config cfg.GosoConf) error {
		if err := config.Option(cfg.WithConfigSetting("dx.use_random_port", true, cfg.SkipExisting)); err != nil {
			return fmt.Errorf("could not set dx.use_random_port: %w", err)
		}

		should, err := ShouldUseRandomPort(config)
		if err != nil {
			return fmt.Errorf("could not check if random port should be used: %w", err)
		}
		if should {
			for setting := range randomizablePortSettings {
				if err := config.Option(cfg.WithConfigSetting(setting, "0", cfg.SkipExisting)); err != nil {
					return fmt.Errorf("could not set %s: %w", setting, err)
				}
			}
		}

		return nil
	})
}

func ShouldUseRandomPort(config cfg.Config) (bool, error) {
	return config.GetBool("dx.use_random_port", false)
}
