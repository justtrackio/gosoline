package mdlsub

import (
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func init() {
	cfg.AddPostProcessor(8, "gosoline.mdlsub.subscriber", SubscriberConfigPostProcessor)
}

type (
	SubscriberInputConfigPostProcessor  func(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) (cfg.Option, error)
	SubscriberOutputConfigPostProcessor func(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) (cfg.Option, error)
)

var subscriberInputConfigPostProcessors = map[string]SubscriberInputConfigPostProcessor{
	stream.InputTypeKafka:   kafkaSubscriberInputConfigPostProcessor,
	stream.InputTypeKinesis: kinesisSubscriberInputConfigPostProcessor,
	stream.InputTypeSns:     snsSubscriberInputConfigPostProcessor,
}

var subscriberOutputConfigPostProcessors = map[string]SubscriberOutputConfigPostProcessor{
	OutputTypeKvstore: kvstoreSubscriberOutputConfigPostProcessor,
}

func SubscriberConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if !config.IsSet(ConfigKeyMdlSubSubscribers) {
		return false, nil
	}

	var ok bool
	var err error
	var consumerName string
	var inputPostProcessor SubscriberInputConfigPostProcessor
	var outputPostProcessor SubscriberOutputConfigPostProcessor
	var inputOption, outputOption cfg.Option

	settings, err := unmarshalSettings(config)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal mdlsub settings: %w", err)
	}

	for name, subscriberSettings := range settings.Subscribers {
		subscriberKey := GetSubscriberConfigKey(name)

		consumerSettings := &stream.ConsumerSettings{}
		if err = config.UnmarshalDefaults(consumerSettings); err != nil {
			return false, fmt.Errorf("can not unmarshal consumer settings for subscriber %s: %w", name, err)
		}

		if consumerSettings.Input, err = GetSubscriberFQN(config, name, subscriberSettings.SourceModel); err != nil {
			return false, fmt.Errorf("can not get subscriber fqn for subscriber %s: %w", name, err)
		}

		if consumerName, err = GetSubscriberFQN(config, name, subscriberSettings.SourceModel); err != nil {
			return false, fmt.Errorf("can not get subscriber fqn for subscriber %s: %w", name, err)
		}

		consumerKey := stream.ConfigurableConsumerKey(consumerName)

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(consumerKey, consumerSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(subscriberKey, subscriberSettings),
		}

		if inputPostProcessor, ok = subscriberInputConfigPostProcessors[subscriberSettings.Input]; ok {
			if inputOption, err = inputPostProcessor(config, name, subscriberSettings); err != nil {
				return false, fmt.Errorf("can not process input config for subscriber %s: %w", name, err)
			}

			configOptions = append(configOptions, inputOption)
		}

		if outputPostProcessor, ok = subscriberOutputConfigPostProcessors[subscriberSettings.Output]; ok {
			if outputOption, err = outputPostProcessor(config, name, subscriberSettings); err != nil {
				return false, fmt.Errorf("can not process output config for subscriber %s: %w", name, err)
			}

			configOptions = append(configOptions, outputOption)
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for subscriber %s: %w", name, err)
		}
	}

	return true, nil
}

func snsSubscriberInputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) (cfg.Option, error) {
	var err error
	var inputKey string

	sourceModel := subscriberSettings.SourceModel
	if inputKey, err = getInputConfigKey(config, name, sourceModel); err != nil {
		return nil, fmt.Errorf("can not get input key for subscriber %s: %w", name, err)
	}

	consumerId := sourceModel.Name
	topicId := sourceModel.Name

	if sourceModel.Shared {
		topicId = sharedName
	}

	if subscriberSettings.TargetModel.Shared {
		consumerId = sharedName
	}

	inputSettings := &stream.SnsInputConfiguration{}
	if err := config.UnmarshalDefaults(inputSettings); err != nil {
		return cfg.WithConfigSetting(inputKey, nil, cfg.SkipExisting), nil
	}

	inputSettings.ConsumerId = consumerId
	inputSettings.Targets = []stream.SnsInputTargetConfiguration{
		{
			Identity: cfg.AppIdentity{
				Env:  sourceModel.ModelId.Env,
				Name: sourceModel.ModelId.App,
				Tags: cfg.AppTags(sourceModel.ModelId.Tags),
			},
			TopicId: topicId,
		},
	}

	return cfg.WithConfigSetting(inputKey, inputSettings, cfg.SkipExisting), nil
}

func kafkaSubscriberInputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) (cfg.Option, error) {
	var err error
	var inputKey string

	sourceModel := subscriberSettings.SourceModel
	if inputKey, err = getInputConfigKey(config, name, sourceModel); err != nil {
		return nil, fmt.Errorf("can not get input key for subscriber %s: %w", name, err)
	}

	topicId := sourceModel.Name
	if sourceModel.Shared {
		topicId = sharedName
	}

	inputSettings := &stream.KafkaInputConfiguration{}
	if err := config.UnmarshalDefaults(inputSettings); err != nil {
		return cfg.WithConfigSetting(inputKey, nil, cfg.SkipExisting), nil
	}

	inputSettings.Tags = sourceModel.Tags
	inputSettings.Name = sourceModel.App
	inputSettings.GroupId = topicId
	inputSettings.TopicId = topicId

	return cfg.WithConfigSetting(inputKey, inputSettings, cfg.SkipExisting), nil
}

func kinesisSubscriberInputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) (cfg.Option, error) {
	var err error
	var inputKey string

	sourceModel := subscriberSettings.SourceModel
	if inputKey, err = getInputConfigKey(config, name, sourceModel); err != nil {
		return nil, fmt.Errorf("can not get input key for subscriber %s: %w", name, err)
	}

	streamName := sourceModel.Name
	if sourceModel.Shared {
		streamName = sharedName
	}

	inputSettings := &stream.KinesisInputConfiguration{}
	if err := config.UnmarshalDefaults(inputSettings); err != nil {
		return cfg.WithConfigSetting(inputKey, nil, cfg.SkipExisting), nil
	}

	inputSettings.Tags = sourceModel.Tags
	inputSettings.Name = sourceModel.Name
	inputSettings.StreamName = streamName

	return cfg.WithConfigSetting(inputKey, inputSettings, cfg.SkipExisting), nil
}

func kvstoreSubscriberOutputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) (cfg.Option, error) {
	kvstoreKey := kvstore.GetConfigurableKey(name)

	kvstoreSettings := &kvstore.ChainConfiguration{}
	if err := config.UnmarshalDefaults(kvstoreSettings); err != nil {
		return cfg.WithConfigSetting(kvstoreKey, nil, cfg.SkipExisting), nil
	}

	// Pad the ModelId from config to fill in any missing fields
	modelId := subscriberSettings.TargetModel.ModelId
	if err := modelId.PadFromConfig(config); err != nil {
		return cfg.WithConfigSetting(kvstoreKey, nil, cfg.SkipExisting), nil
	}

	kvstoreSettings.ModelId = modelId
	kvstoreSettings.Elements = []string{kvstore.TypeRedis, kvstore.TypeDdb}

	return cfg.WithConfigSetting(kvstoreKey, kvstoreSettings, cfg.SkipExisting), nil
}

func GetSubscriberFQN(config cfg.Config, name string, sourceModel SubscriberModel) (string, error) {
	if !sourceModel.Shared {
		return fmt.Sprintf("subscriber-%s", name), nil
	}

	domain := sourceModel.DomainString()
	domain = strings.Replace(domain, ".", "_", -1)

	return fmt.Sprintf("subscriber-%s-%s", domain, sharedName), nil
}

func getInputConfigKey(config cfg.Config, name string, sourceModel SubscriberModel) (string, error) {
	var err error
	var inputName string

	if inputName, err = GetSubscriberFQN(config, name, sourceModel); err != nil {
		return "", fmt.Errorf("failed to get subscriber fqn: %w", err)
	}

	return stream.ConfigurableInputKey(inputName), nil
}

func GetSubscriberConfigKey(name string) string {
	return fmt.Sprintf("%s.%s", ConfigKeyMdlSubSubscribers, name)
}

func GetSubscriberOutputConfigKey(name string) string {
	return fmt.Sprintf("%s.output", GetSubscriberConfigKey(name))
}

func UnmarshalSubscriberSourceModel(config cfg.Config, name string) (SubscriberModel, error) {
	key := fmt.Sprintf("%s.source", GetSubscriberConfigKey(name))
	sourceModel := &SubscriberModel{}
	if err := config.UnmarshalKey(key, sourceModel); err != nil {
		return SubscriberModel{}, fmt.Errorf("failed to unmarshal subscriber source model: %w", err)
	}

	if sourceModel.Name == "" {
		sourceModel.Name = name
	}

	return *sourceModel, nil
}
