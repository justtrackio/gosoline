package stream

import (
	"fmt"

	"github.com/applike/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.stream.producerDaemon", producerDaemonConfigPostprocessor)
}

func producerDaemonConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	producerDaemonSettings := readAllProducerDaemonSettings(config)

	if len(producerDaemonSettings) == 0 {
		return false, nil
	}

	for name, settings := range producerDaemonSettings {
		outputKey := ConfigurableOutputKey(settings.Output)
		outputSettings := &BaseOutputSettings{}

		config.UnmarshalKey(outputKey, outputSettings)
		outputSettings.Tracing.Enabled = false

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(outputKey, outputSettings),
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for producer daemon %s: %w", name, err)
		}
	}

	return true, nil
}
