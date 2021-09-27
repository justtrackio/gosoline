package dx

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(1, "gosoline.dx.usePathStyle", UsePathStyleConfigPostProcessor)
}

func UsePathStyleConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	return runPostProcessorForDev(config, func(config cfg.GosoConf) error {
		if err := config.Option(cfg.WithConfigSetting("cloud.aws.s3.clients.default.usePathStyle", true, cfg.SkipExisting)); err != nil {
			return fmt.Errorf("could not set cloud.aws.s3.clients.default.usePathStyle: %w", err)
		}

		return nil
	})
}
