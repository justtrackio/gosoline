package mdlsub

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func init() {
	cfg.AddPostProcessor(8, "gosoline.mdlsub.publisher", PublisherConfigPostProcessor)
}

const sharedName = "publisher"

type publisherOutputTypeHandler func(config cfg.Config, publisherSettings *PublisherSettings, producerSettings *stream.ProducerSettings, clientName string) stream.BaseOutputConfigurationAware

var publisherOutputTypeHandlers = map[string]publisherOutputTypeHandler{
	stream.OutputTypeInMemory: handlePublisherOutputTypeInMemory,
	stream.OutputTypeKinesis:  handlePublisherOutputTypeKinesis,
	stream.OutputTypeSns:      handlePublisherOutputTypeSns,
	stream.OutputTypeSqs:      handlePublisherOutputTypeSqs,
}

func PublisherConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if !config.IsSet(ConfigKeyMdlSubPublishers) {
		return false, nil
	}

	publishers := config.GetStringMap(ConfigKeyMdlSubPublishers)

	for name := range publishers {
		publisherKey := getPublisherConfigKey(name)
		publisherSettings := readPublisherSetting(config, name)

		producerName := fmt.Sprintf("publisher-%s", name)
		outputName := fmt.Sprintf("publisher-%s", name)

		if len(publisherSettings.Producer) != 0 {
			producerName = publisherSettings.Producer
		} else {
			publisherSettings.Producer = producerName
		}

		producerSettings := &stream.ProducerSettings{}
		config.UnmarshalDefaults(producerSettings)

		producerSettings.Output = outputName
		producerSettings.Daemon.MessageAttributes[AttributeModelId] = publisherSettings.ModelId.String()

		var ok bool
		var outputTypeHandler publisherOutputTypeHandler

		if outputTypeHandler, ok = publisherOutputTypeHandlers[publisherSettings.OutputType]; !ok {
			return false, fmt.Errorf("no publisherOutputTypeHandler found for publisher %s and output type %s", publisherSettings.Name, publisherSettings.OutputType)
		}

		clientName := producerName
		outputSettings := outputTypeHandler(config, publisherSettings, producerSettings, clientName)
		producerKey := stream.ConfigurableProducerKey(producerName)
		outputKey := stream.ConfigurableOutputKey(outputName)

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(producerKey, producerSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(outputKey, outputSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(publisherKey, publisherSettings),
		}

		if producerSettings.Daemon.Enabled {
			// if the producer daemon is enabled, default to infinite retries for it.
			// otherwise, if you have an API or similar, we will only retry for a time
			// fitting the request timeout, but the producer daemon runs in the background,
			// so it isn't bound like this
			awsClientKey := aws.GetDefaultsKey(clientName) + ".backoff.type"
			configOptions = append(configOptions, cfg.WithConfigSetting(awsClientKey, "infinite", cfg.SkipExisting))
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for publisher %s: %w", publisherSettings.Name, err)
		}
	}

	return true, nil
}

func handlePublisherOutputTypeInMemory(config cfg.Config, _ *PublisherSettings, _ *stream.ProducerSettings, _ string) stream.BaseOutputConfigurationAware {
	outputSettings := &stream.InMemoryOutputConfiguration{}
	config.UnmarshalDefaults(outputSettings)

	return outputSettings
}

func handlePublisherOutputTypeKinesis(config cfg.Config, publisherSettings *PublisherSettings, producerSettings *stream.ProducerSettings, clientName string) stream.BaseOutputConfigurationAware {
	producerSettings.Daemon.Enabled = true
	producerSettings.Daemon.Interval = time.Second
	// kinesis batches have a max size of 5mb. we're using 4.5mb to give it some headroom
	producerSettings.Daemon.BatchMaxSize = 4_500_000
	// kinesis can handle up to 500 records per put records call
	producerSettings.Daemon.BatchSize = 500
	// kinesis limit for 1 record in size is 1mb, so we limit it to 950kb to give it some headroom
	producerSettings.Daemon.AggregationMaxSize = 950_000

	outputSettings := &stream.KinesisOutputConfiguration{}
	config.UnmarshalDefaults(outputSettings)

	outputSettings.Project = publisherSettings.Project
	outputSettings.Family = publisherSettings.Family
	outputSettings.Group = publisherSettings.Group
	outputSettings.Application = publisherSettings.Application
	outputSettings.ClientName = clientName
	outputSettings.StreamName = publisherSettings.Name
	outputSettings.Tracing.Enabled = false

	return outputSettings
}

func handlePublisherOutputTypeSns(config cfg.Config, publisherSettings *PublisherSettings, _ *stream.ProducerSettings, clientName string) stream.BaseOutputConfigurationAware {
	outputSettings := &stream.SnsOutputConfiguration{}
	config.UnmarshalDefaults(outputSettings)

	outputSettings.Project = publisherSettings.Project
	outputSettings.Family = publisherSettings.Family
	outputSettings.Group = publisherSettings.Group
	outputSettings.Application = publisherSettings.Application
	outputSettings.TopicId = publisherSettings.Name
	outputSettings.ClientName = clientName

	if publisherSettings.Shared {
		outputSettings.TopicId = sharedName
	}

	return outputSettings
}

func handlePublisherOutputTypeSqs(config cfg.Config, publisherSettings *PublisherSettings, _ *stream.ProducerSettings, clientName string) stream.BaseOutputConfigurationAware {
	outputSettings := &stream.SqsOutputConfiguration{}
	config.UnmarshalDefaults(outputSettings)

	outputSettings.Project = publisherSettings.Project
	outputSettings.Family = publisherSettings.Family
	outputSettings.Group = publisherSettings.Group
	outputSettings.Application = publisherSettings.Application
	outputSettings.QueueId = publisherSettings.Name
	outputSettings.ClientName = clientName

	if publisherSettings.Shared {
		outputSettings.QueueId = sharedName
	}

	return outputSettings
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
