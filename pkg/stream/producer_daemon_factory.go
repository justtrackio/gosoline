package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

func ProducerDaemonFactory(config cfg.Config, logger mon.Logger) (map[string]kernel.Module, error) {
	modules := map[string]kernel.Module{}
	producerDaemonSettings := readAllProducerDaemonSettings(config)

	for name, settings := range producerDaemonSettings {
		if !settings.Daemon.Enabled {
			continue
		}

		moduleName := fmt.Sprintf("producer-daemon-%s", name)
		modules[moduleName] = ProvideProducerDaemon(config, logger, name)
	}

	return modules, nil
}
