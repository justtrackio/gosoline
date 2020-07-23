package pubsub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/stream"
)

func init() {
	cfg.AddPostProcessor(8, "gosoline.pubsub.subscriber", SubscriberConfigPostProcessor)
}

type SubscriberInputConfigPostProcessor func(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option
type SubscriberOutputConfigPostProcessor func(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option

var subscriberInputConfigPostProcessors = map[string]SubscriberInputConfigPostProcessor{
	"sns": snsSubscriberInputConfigPostProcessor,
}

var subscriberOutputConfigPostProcessors = map[string]SubscriberOutputConfigPostProcessor{
	"kvstore": kvstoreSubscriberOutputConfigPostProcessor,
}

func SubscriberConfigPostProcessor(config cfg.GosoConf) (bool, error) {
	if !config.IsSet(ConfigKeyPubSubSubscribers) {
		return false, nil
	}

	var ok bool
	var inputPostProcessor SubscriberInputConfigPostProcessor
	var outputPostProcessor SubscriberOutputConfigPostProcessor

	subscriberSettingsMap := make(map[string]*SubscriberSettings)
	config.UnmarshalKey(ConfigKeyPubSubSubscribers, &subscriberSettingsMap)

	for name, subscriberSettings := range subscriberSettingsMap {
		subscriberKey := getSubscriberConfigKey(name)

		if subscriberSettings.SourceModel.Name == "" {
			subscriberSettings.SourceModel.Name = name
		}

		if subscriberSettings.TargetModel.Name == "" {
			subscriberSettings.TargetModel.Name = name
		}

		consumerName := fmt.Sprintf("subscriber-%s", name)
		consumerKey := stream.ConfigurableConsumerKey(consumerName)

		consumerSettings := &stream.ConsumerSettings{}
		config.UnmarshalDefaults(consumerSettings)

		consumerSettings.Input = getInputName(name)

		if inputPostProcessor, ok = subscriberInputConfigPostProcessors[subscriberSettings.Input]; !ok {
			return false, fmt.Errorf("there is no SubscriberInputConfigPostProcessor for input %s", subscriberSettings.Input)
		}

		inputOption := inputPostProcessor(config, name, subscriberSettings)

		if outputPostProcessor, ok = subscriberOutputConfigPostProcessors[subscriberSettings.Output]; !ok {
			return false, fmt.Errorf("there is no subscriberOutputConfigPostProcessors for output %s", subscriberSettings.Output)
		}

		outputOption := outputPostProcessor(config, name, subscriberSettings)

		configOptions := []cfg.Option{
			cfg.WithConfigSetting(consumerKey, consumerSettings, cfg.MergeWithoutOverride),
			cfg.WithConfigSetting(subscriberKey, subscriberSettings),
			inputOption,
			outputOption,
		}

		if err := config.Option(configOptions...); err != nil {
			return false, fmt.Errorf("can not apply config settings for subscriber %s: %w", name, err)
		}
	}

	return true, nil
}

func snsSubscriberInputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option {
	inputKey := getInputConfigKey(name)

	inputSettings := &stream.SnsInputConfiguration{}
	config.UnmarshalDefaults(inputSettings, cfg.UnmarshalWithDefaultsFromKey(stream.ConfigKeyStreamBackoff, "backoff"))

	inputSettings.ConsumerId = subscriberSettings.SourceModel.Name
	inputSettings.Targets = []stream.SnsInputTargetConfiguration{
		{
			Family:      subscriberSettings.SourceModel.Family,
			Application: subscriberSettings.SourceModel.Application,
			TopicId:     subscriberSettings.SourceModel.Name,
		},
	}

	return cfg.WithConfigSetting(inputKey, inputSettings, cfg.MergeWithoutOverride)
}

func kvstoreSubscriberOutputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option {
	kvstoreName := fmt.Sprintf("subscriber-%s", name)
	kvstoreKey := kvstore.GetConfigurableKey(kvstoreName)

	kvstoreSettings := &kvstore.ChainConfiguration{}
	config.UnmarshalDefaults(kvstoreSettings)

	kvstoreSettings.Project = subscriberSettings.TargetModel.Project
	kvstoreSettings.Family = subscriberSettings.TargetModel.Family
	kvstoreSettings.Application = subscriberSettings.TargetModel.Application
	kvstoreSettings.Elements = []string{kvstore.TypeRedis, kvstore.TypeDdb}

	return cfg.WithConfigSetting(kvstoreKey, kvstoreSettings, cfg.MergeWithoutOverride)
}

func getInputName(name string) string {
	return fmt.Sprintf("subscriber-%s", name)
}

func getInputConfigKey(name string) string {
	inputName := getInputName(name)

	return stream.ConfigurableInputKey(inputName)
}

func getSubscriberConfigKey(name string) string {
	return fmt.Sprintf("%s.%s", ConfigKeyPubSubSubscribers, name)
}
