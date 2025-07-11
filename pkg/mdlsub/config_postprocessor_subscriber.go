package mdlsub

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func init() {
	cfg.AddPostProcessor(8, "gosoline.mdlsub.subscriber", SubscriberConfigPostProcessor)
}

type (
	SubscriberInputConfigPostProcessor  func(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option
	SubscriberOutputConfigPostProcessor func(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option
)

var subscriberInputConfigPostProcessors = map[string]SubscriberInputConfigPostProcessor{
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
	var inputPostProcessor SubscriberInputConfigPostProcessor
	var outputPostProcessor SubscriberOutputConfigPostProcessor

	settings, err := unmarshalSettings(config)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal mdlsub settings: %w", err)
	}

	for name, subscriberSettings := range settings.Subscribers {
		subscriberKey := GetSubscriberConfigKey(name)

		consumerSettings := &stream.ConsumerSettings{}
		if err := config.UnmarshalDefaults(consumerSettings); err != nil {
			return false, fmt.Errorf("can not unmarshal consumer settings for subscriber %s: %w", name, err)
		}

		consumerSettings.Input = GetSubscriberFQN(name, subscriberSettings.SourceModel)
		consumerName := GetSubscriberFQN(name, subscriberSettings.SourceModel)
		consumerKey := stream.ConfigurableConsumerKey(consumerName)

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(consumerKey, consumerSettings, cfg.SkipExisting),
			cfg.WithConfigSetting(subscriberKey, subscriberSettings),
		}

		if inputPostProcessor, ok = subscriberInputConfigPostProcessors[subscriberSettings.Input]; ok {
			inputOption := inputPostProcessor(config, name, subscriberSettings)
			configOptions = append(configOptions, inputOption)
		}

		if outputPostProcessor, ok = subscriberOutputConfigPostProcessors[subscriberSettings.Output]; ok {
			outputOption := outputPostProcessor(config, name, subscriberSettings)
			configOptions = append(configOptions, outputOption)
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for subscriber %s: %w", name, err)
		}
	}

	return true, nil
}

func snsSubscriberInputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option {
	inputKey := getInputConfigKey(name, subscriberSettings.SourceModel)
	consumerId := subscriberSettings.SourceModel.Name
	topicId := subscriberSettings.SourceModel.Name

	if subscriberSettings.SourceModel.Shared {
		topicId = sharedName
	}

	if subscriberSettings.TargetModel.Shared {
		consumerId = sharedName
	}

	inputSettings := &stream.SnsInputConfiguration{}
	if err := config.UnmarshalDefaults(inputSettings); err != nil {
		return cfg.WithConfigSetting(inputKey, nil, cfg.SkipExisting)
	}

	inputSettings.ConsumerId = consumerId
	inputSettings.Targets = []stream.SnsInputTargetConfiguration{
		{
			Family:      subscriberSettings.SourceModel.Family,
			Group:       subscriberSettings.SourceModel.Group,
			Application: subscriberSettings.SourceModel.Application,
			TopicId:     topicId,
		},
	}

	return cfg.WithConfigSetting(inputKey, inputSettings, cfg.SkipExisting)
}

func kinesisSubscriberInputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option {
	inputKey := getInputConfigKey(name, subscriberSettings.SourceModel)
	streamName := subscriberSettings.SourceModel.Name

	if subscriberSettings.SourceModel.Shared {
		streamName = sharedName
	}

	inputSettings := &stream.KinesisInputConfiguration{}
	if err := config.UnmarshalDefaults(inputSettings); err != nil {
		return cfg.WithConfigSetting(inputKey, nil, cfg.SkipExisting)
	}

	inputSettings.Project = subscriberSettings.SourceModel.Project
	inputSettings.Family = subscriberSettings.SourceModel.Family
	inputSettings.Group = subscriberSettings.SourceModel.Group
	inputSettings.Application = subscriberSettings.SourceModel.Application
	inputSettings.StreamName = streamName

	return cfg.WithConfigSetting(inputKey, inputSettings, cfg.SkipExisting)
}

func kvstoreSubscriberOutputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option {
	kvstoreKey := kvstore.GetConfigurableKey(name)

	kvstoreSettings := &kvstore.ChainConfiguration{}
	if err := config.UnmarshalDefaults(kvstoreSettings); err != nil {
		return cfg.WithConfigSetting(kvstoreKey, nil, cfg.SkipExisting)
	}

	kvstoreSettings.Project = subscriberSettings.TargetModel.Project
	kvstoreSettings.Family = subscriberSettings.TargetModel.Family
	kvstoreSettings.Group = subscriberSettings.TargetModel.Group
	kvstoreSettings.Application = subscriberSettings.TargetModel.Application
	kvstoreSettings.Elements = []string{kvstore.TypeRedis, kvstore.TypeDdb}

	return cfg.WithConfigSetting(kvstoreKey, kvstoreSettings, cfg.SkipExisting)
}

func GetSubscriberFQN(name string, sourceModel SubscriberModel) string {
	if !sourceModel.Shared {
		return fmt.Sprintf("subscriber-%s", name)
	}

	return fmt.Sprintf("subscriber-%s-%s-%s-%s-%s", sourceModel.Project, sourceModel.Family, sourceModel.Group, sourceModel.Application, sharedName)
}

func getInputConfigKey(name string, sourceModel SubscriberModel) string {
	inputName := GetSubscriberFQN(name, sourceModel)

	return stream.ConfigurableInputKey(inputName)
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
