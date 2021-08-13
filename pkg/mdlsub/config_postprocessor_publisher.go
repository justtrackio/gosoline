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
		publisherSettings := readPublisherSetting(config, name)

		outputSettings := &stream.SnsOutputConfiguration{}
		config.UnmarshalDefaults(outputSettings)

		outputSettings.Type = publisherSettings.OutputType
		outputSettings.Project = publisherSettings.Project
		outputSettings.Family = publisherSettings.Family
		outputSettings.Application = publisherSettings.Application
		outputSettings.TopicId = publisherSettings.Name

		if publisherSettings.Shared {
			outputSettings.TopicId = "publisher"
		}

		producerName := fmt.Sprintf("publisher-%s", publisherSettings.Name)
		outputName := fmt.Sprintf("publisher-%s", publisherSettings.Name)

		if len(publisherSettings.Producer) != 0 {
			producerName = publisherSettings.Producer
		} else {
			publisherSettings.Producer = producerName
		}

		producerSettings := &stream.ProducerSettings{}
		config.UnmarshalDefaults(producerSettings)

		producerSettings.Output = outputName
		producerSettings.Daemon.MessageAttributes[AttributeModelId] = publisherSettings.ModelId.String()

		producerKey := stream.ConfigurableProducerKey(producerName)
		outputKey := stream.ConfigurableOutputKey(outputName)

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(producerKey, producerSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(outputKey, outputSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(publisherKey, publisherSettings),
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for publisher %s: %w", publisherSettings.Name, err)
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
