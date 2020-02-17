// +build integration

package es_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

func testConfig(filePath string) cfg.Config {
	config := cfg.New()

	err := cfg.WithConfigFile(filePath, "yml")(config)

	if err != nil {
		panic(err)
	}

	return config
}

func getMocks(configFilePath string) (cfg.Config, mon.Logger) {
	config := testConfig(configFilePath)
	logger := mon.NewLogger()

	return config, logger
}
