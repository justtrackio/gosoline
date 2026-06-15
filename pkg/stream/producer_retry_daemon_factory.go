package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func ProducerRetryDaemonFactory(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
	modules := map[string]kernel.ModuleFactory{}
	producerRetrySettings, err := readAllProducerRetrySettings(config)
	if err != nil {
		return nil, fmt.Errorf("can not read all producer retry settings: %w", err)
	}

	for name, settings := range producerRetrySettings {
		daemon, err := ProvideProducerRetryDaemon(ctx, config, logger, settings.Output, RetryMetadata{
			name:           name,
			retryConfigKey: ConfigurableProducerRetryKey(name),
			retrySettings:  &settings.Retry,
		})
		if err != nil {
			return nil, fmt.Errorf("can not create producer daemon %s: %w", name, err)
		}

		moduleName := fmt.Sprintf("producer-daemon-%s", name)
		modules[moduleName] = func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return daemon, nil
		}
	}

	return modules, nil
}

func readAllProducerRetrySettings(config cfg.Config) (map[string]*ProducerSettings, error) {
	producerSettings := make(map[string]*ProducerSettings)
	producerMap, err := config.GetStringMap("stream.producer", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("can not get producer map: %w", err)
	}

	for name := range producerMap {
		settings, err := readProducerSettings(config, name)
		if err != nil {
			return nil, fmt.Errorf("can not read producer settings for %s: %w", name, err)
		}
		if !settings.Retry.Enabled {
			continue
		}

		producerSettings[name] = settings
	}

	return producerSettings, nil
}
