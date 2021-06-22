package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/test/env"
)

func Boot(configFilenames ...string) (*Mocks, error) {
	config := cfg.New()
	logger, err := env.NewConsoleLogger()

	if err != nil {
		return nil, fmt.Errorf("could not create console logger: %w", err)
	}

	if len(configFilenames) == 0 {
		configFilenames = []string{"config.test.yml"}
	}

	for _, filename := range configFilenames {
		err := config.Option(cfg.WithConfigFile(filename, "yml"))

		if err != nil {
			return nil, fmt.Errorf("failed to read config from file %s: %w", filename, err)
		}
	}

	err = config.Option(cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer))
	if err != nil {
		return nil, fmt.Errorf("failed to set env key replacer: %w", err)
	}

	return newMocks(config, logger)
}
