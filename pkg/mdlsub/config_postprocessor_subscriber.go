package mdlsub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/stream"
)

func init() {
	cfg.AddPostProcessor(8, "gosoline.mdlsub.subscriber", SubscriberConfigPostProcessor)
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
	if !config.IsSet(ConfigKeyMdlSubSubscribers) {
		return false, nil
	}

	var ok bool
	var inputPostProcessor SubscriberInputConfigPostProcessor
	var outputPostProcessor SubscriberOutputConfigPostProcessor

	subscriberSettingsMap := make(map[string]*SubscriberSettings)
	config.UnmarshalKey(ConfigKeyMdlSubSubscribers, &subscriberSettingsMap)

	for name, subscriberSettings := range subscriberSettingsMap {
		subscriberKey := GetSubscriberConfigKey(name)

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

	return cfg.WithConfigSetting(inputKey, inputSettings, cfg.SkipExisting)
}

func kvstoreSubscriberOutputConfigPostProcessor(config cfg.GosoConf, name string, subscriberSettings *SubscriberSettings) cfg.Option {
	kvstoreKey := kvstore.GetConfigurableKey(name)

	kvstoreSettings := &kvstore.Configuration{}
	config.UnmarshalDefaults(kvstoreSettings)

	kvstoreSettings.Project = subscriberSettings.TargetModel.Project
	kvstoreSettings.Family = subscriberSettings.TargetModel.Family
	kvstoreSettings.Application = subscriberSettings.TargetModel.Application
	kvstoreSettings.Elements = []string{kvstore.TypeRedis, kvstore.TypeDdb}

	return cfg.WithConfigSetting(kvstoreKey, kvstoreSettings, cfg.SkipExisting)
}

func getInputName(name string) string {
	return fmt.Sprintf("subscriber-%s", name)
}

func getInputConfigKey(name string) string {
	inputName := getInputName(name)

	return stream.ConfigurableInputKey(inputName)
}

func GetSubscriberConfigKey(name string) string {
	return fmt.Sprintf("%s.%s", ConfigKeyMdlSubSubscribers, name)
}

func GetSubscriberOutputConfigKey(name string) string {
	return fmt.Sprintf("%s.output", GetSubscriberConfigKey(name))
}
