package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

func Boot(configFilenames ...string) (*Mocks, error) {
	config := cfg.New()
	logger := mon.NewLogger()

	if len(configFilenames) == 0 {
		configFilenames = []string{"config.test.yml"}
	}

	for _, filename := range configFilenames {
		err := config.Option(cfg.WithConfigFile(filename, "yml"))

		if err != nil {
			return nil, fmt.Errorf("failed to read config from file %s: %w", filename, err)
		}
	}

	return newMocks(config, logger)
}
