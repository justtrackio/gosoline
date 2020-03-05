// +build integration

package es_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

func getMocks(configFilePath string) (cfg.Config, mon.Logger) {
	config := cfg.New()
	config.Option(cfg.WithConfigFile(configFilePath, "yml"))
	logger := mon.NewLogger()

	return config, logger
}
