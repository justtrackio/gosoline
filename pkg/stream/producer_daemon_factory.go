package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

func ProducerDaemonFactory(config cfg.Config, logger mon.Logger) (map[string]kernel.ModuleFactory, error) {
	modules := map[string]kernel.ModuleFactory{}
	producerDaemonSettings := readAllProducerDaemonSettings(config)

	for name, settings := range producerDaemonSettings {
		producerName := name

		if !settings.Daemon.Enabled {
			continue
		}

		moduleName := fmt.Sprintf("producer-daemon-%s", producerName)
		modules[moduleName] = func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
			return ProvideProducerDaemon(config, logger, producerName), nil
		}
	}

	return modules, nil
}
