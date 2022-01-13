package mdlsub

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/stream"
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

		outputConfig := &stream.SnsOutputConfiguration{}
		config.UnmarshalDefaults(outputConfig)

		outputConfig.Type = publisherSettings.OutputType
		outputConfig.Project = publisherSettings.Project
		outputConfig.Family = publisherSettings.Family
		outputConfig.Application = publisherSettings.Application
		outputConfig.TopicId = publisherSettings.Name

		if publisherSettings.Shared {
			outputConfig.TopicId = "publisher"
		}

		producerName := fmt.Sprintf("publisher-%s", publisherSettings.Name)
		outputName := fmt.Sprintf("publisher-%s", publisherSettings.Name)

		producerSettings := &stream.ProducerSettings{}
		config.UnmarshalDefaults(producerSettings)

		producerSettings.Output = outputName
		producerSettings.Daemon.MessageAttributes[AttributeModelId] = publisherSettings.ModelId.String()

		var outputSettings interface{}
		outputSettings = outputConfig

		if len(publisherSettings.Producer) != 0 {
			producerName = publisherSettings.Producer
			outputSettings = producerSettings.Output
		} else {
			publisherSettings.Producer = producerName
		}

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
