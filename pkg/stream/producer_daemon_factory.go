package stream

import (
	"context"
	"fmt"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
)

func ProducerDaemonFactory(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
	modules := map[string]kernel.ModuleFactory{}
	producerDaemonSettings := readAllProducerDaemonSettings(config)

	for name, settings := range producerDaemonSettings {
		if !settings.Daemon.Enabled {
			continue
		}

		if daemon, err := ProvideProducerDaemon(ctx, config, logger, name); err != nil {
			return nil, fmt.Errorf("can not create producer daemon %s: %w", name, err)
		} else {
			moduleName := fmt.Sprintf("producer-daemon-%s", name)
			modules[moduleName] = func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
				return daemon, nil
			}
		}
	}

	return modules, nil
}
