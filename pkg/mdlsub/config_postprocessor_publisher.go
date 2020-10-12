package mdlsub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/stream"
)

func init() {
	cfg.AddPostProcessor(8, "gosoline.mdlsub.publisher", PublisherConfigPostProcessor)
}

func PublisherConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if !config.IsSet(ConfigKeyMdlSubPublishers) {
		return false, nil
	}

	publishers := config.GetStringMap(ConfigKeyMdlSubPublishers)

	for name := range publishers {
		publisherKey := getPublisherConfigKey(name)
		settings := readPublisherSetting(config, name)

		outputSettings := &stream.SnsOutputConfiguration{}
		config.UnmarshalDefaults(outputSettings, cfg.UnmarshalWithDefaultsFromKey(stream.ConfigKeyStreamBackoff, "backoff"))

		outputSettings.Type = settings.OutputType
		outputSettings.Project = settings.Project
		outputSettings.Family = settings.Family
		outputSettings.Application = settings.Application
		outputSettings.TopicId = settings.Name

		if settings.Shared {
			outputSettings.TopicId = "publisher"
		}

		producerName := fmt.Sprintf("publisher-%s", settings.Name)
		outputName := fmt.Sprintf("publisher-%s", settings.Name)

		if len(settings.Producer) != 0 {
			producerName = settings.Producer
		} else {
			settings.Producer = producerName
		}

		producerSettings := &stream.ProducerSettings{}
		config.UnmarshalDefaults(producerSettings)

		producerSettings.Output = outputName

		producerKey := stream.ConfigurableProducerKey(producerName)
		outputKey := stream.ConfigurableOutputKey(outputName)

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(producerKey, producerSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(outputKey, outputSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(publisherKey, settings),
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for publisher %s: %w", settings.Name, err)
		}
	}

	return true, nil
}

func getPublisherConfigKey(name string) string {
	return fmt.Sprintf("%s.%s", ConfigKeyMdlSubPublishers, name)
}

func readPublisherSetting(config cfg.Config, name string) *PublisherSettings {
	publisherKey := getPublisherConfigKey(name)

	settings := &PublisherSettings{}
	config.UnmarshalKey(publisherKey, settings)

	if settings.Name == "" {
		settings.Name = name
	}

	return settings
}
