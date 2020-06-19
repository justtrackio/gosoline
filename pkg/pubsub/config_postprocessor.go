package pubsub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/stream"
)

func ConfigPostProcessor(config cfg.GosoConf) error {
	publisherSettings := readPublisherSettings(config)

	for i, settings := range publisherSettings {
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
		publisherKey := fmt.Sprintf("%s[%d]", ConfigKeyPubSubPublishers, i)

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(producerKey, producerSettings, cfg.MergeWithoutOverride),
			cfg.WithConfigSetting(outputKey, outputSettings, cfg.MergeWithoutOverride),
			cfg.WithConfigSetting(publisherKey, settings),
		}

		if err := config.Option(configOptions...); err != nil {
			return fmt.Errorf("can not apply config settings for publisher %s: %w", settings.Name, err)
		}
	}

	return nil
}

func readPublisherSettings(config cfg.Config) []*PublisherSettings {
	publisherSettings := make([]*PublisherSettings, 0)
	config.UnmarshalKey(ConfigKeyPubSubPublishers, &publisherSettings)

	return publisherSettings
}
