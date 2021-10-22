package dx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(1, "gosoline.dx.useLocalstackDefaults", UseLocalstackDefaultsConfigPostProcessor)
}

var localstackSetting = make(map[string]interface{})

func RegisterLocalstackSetting(setting string, value interface{}) {
	localstackSetting[setting] = value
}

func UseLocalstackDefaultsConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	return runPostProcessorForDev(config, func(config cfg.GosoConf) error {
		if err := config.Option(cfg.WithConfigSetting("dx.use_localstack_defaults", true, cfg.SkipExisting)); err != nil {
			return fmt.Errorf("could not set dx.use_localstack_defaults: %w", err)
		}

		if ShouldUseLocalstackDefaults(config) {
			for setting, value := range localstackSetting {
				if err := config.Option(cfg.WithConfigSetting(setting, value, cfg.SkipExisting)); err != nil {
					return fmt.Errorf("could not set %s to %v: %w", setting, value, err)
				}
			}
		}

		return nil
	})
}

func ShouldUseLocalstackDefaults(config cfg.Config) bool {
	return config.GetBool("dx.use_localstack_defaults", false)
}
