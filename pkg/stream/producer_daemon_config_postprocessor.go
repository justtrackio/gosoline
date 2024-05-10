package stream

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	cfg.AddPostProcessor(16, "gosoline.stream.producerDaemon", producerDaemonConfigPostprocessor)
}

func producerDaemonConfigPostprocessor(config cfg.GosoConf) (bool, error) {
	producerDaemonSettings, err := readAllProducerDaemonSettings(config)
	if err != nil {
		return false, fmt.Errorf("failed to read all producer daemon settings in producerDaemonConfigPostprocessor: %w", err)
	}

	if len(producerDaemonSettings) == 0 {
		return false, nil
	}

	for name, settings := range producerDaemonSettings {
		outputKey := ConfigurableOutputKey(settings.Output)
		outputSettings := &BaseOutputConfiguration{}

		if err := config.UnmarshalKey(outputKey, outputSettings); err != nil {
			return false, fmt.Errorf("failed to unmarshal output settings for key %q in producerDaemonConfigPostprocessor: %w", outputKey, err)
		}
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
