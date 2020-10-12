package mdlsub

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func NewSubscriberFactory(transformerFactoryMap TransformerMapTypeVersionFactories) kernel.ModuleFactory {
	return func(config cfg.Config, logger mon.Logger) (map[string]kernel.Module, error) {
		return SubscriberFactory(config, logger, transformerFactoryMap)
	}
}

func SubscriberFactory(config cfg.Config, logger mon.Logger, transformerFactories TransformerMapTypeVersionFactories) (map[string]kernel.Module, error) {
	subscriberSettings := make(map[string]*SubscriberSettings)
	config.UnmarshalKey(ConfigKeyMdlSubSubscribers, &subscriberSettings)

	var err error
	var transformers ModelTransformers
	var outputs Outputs

	if transformers, err = initTransformers(config, logger, subscriberSettings, transformerFactories); err != nil {
		return nil, fmt.Errorf("can not create subscribers: %w", err)
	}

	if outputs, err = initOutputs(config, logger, subscriberSettings, transformers); err != nil {
		return nil, fmt.Errorf("can not create subscribers: %w", err)
	}

	var modules = make(map[string]kernel.Module)

	for name := range subscriberSettings {
		moduleName := fmt.Sprintf("subscriber-%s", name)
		consumerName := fmt.Sprintf("subscriber-%s", name)
		callback := NewSubscriberCallback(transformers, outputs)

		modules[moduleName] = stream.NewConsumer(consumerName, callback)
	}

	return modules, nil
}
