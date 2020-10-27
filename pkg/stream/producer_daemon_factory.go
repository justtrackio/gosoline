package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
)

func ProducerDaemonFactory(config cfg.Config, logger mon.Logger) (map[string]kernel.Module, error) {
	modules := map[string]kernel.Module{}
	producerMap := config.GetStringMap("stream.producer", map[string]interface{}{})

	for name := range producerMap {
		key := ConfigurableProducerKey(name)
		settings := &ProducerSettings{}
		config.UnmarshalKey(key, settings)

		if !settings.Daemon.Enabled {
			continue
		}

		moduleName := fmt.Sprintf("producer-daemon-%s", name)
		modules[moduleName] = ProvideProducerDaemon(config, logger, name)
	}

	return modules, nil
}
