package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

const (
	ConfigKeyMdlSub = "mdlsub"
)

type Settings struct {
	SubscriberApi SubscriberApiSettings          `cfg:"subscriber_api"`
	Subscribers   map[string]*SubscriberSettings `cfg:"subscribers"`
}

func NewSubscriberFactory(transformerFactoryMap TransformerMapTypeVersionFactories) kernel.ModuleMultiFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		return SubscriberFactory(ctx, config, logger, transformerFactoryMap)
	}
}

func SubscriberFactory(ctx context.Context, config cfg.Config, logger log.Logger, transformerFactories TransformerMapTypeVersionFactories) (map[string]kernel.ModuleFactory, error) {
	settings := Settings{
		Subscribers: make(map[string]*SubscriberSettings),
	}
	config.UnmarshalKey(fmt.Sprintf("%s.%s", ConfigKeyMdlSub, "subscribers"), &settings.Subscribers)
	config.UnmarshalKey(fmt.Sprintf("%s.%s", ConfigKeyMdlSub, "subscriber_api"), &settings.SubscriberApi)

	var err error
	var transformers ModelTransformers
	var outputs Outputs

	if transformers, err = initTransformers(ctx, config, logger, settings.Subscribers, transformerFactories); err != nil {
		return nil, fmt.Errorf("failed to init transformers: %w", err)
	}

	if outputs, err = initOutputs(ctx, config, logger, settings.Subscribers, transformers); err != nil {
		return nil, fmt.Errorf("failed to init outputs: %w", err)
	}

	modules := make(map[string]kernel.ModuleFactory)

	for name, subscriberSettings := range settings.Subscribers {
		subscriberFQN := GetSubscriberFQN(name, subscriberSettings.SourceModel)

		if _, ok := modules[subscriberFQN]; ok {
			continue
		}

		callbackFactory := NewSubscriberCallbackFactory(transformers, outputs)
		modules[subscriberFQN] = stream.NewConsumer(subscriberFQN, callbackFactory)
	}

	if !settings.SubscriberApi.Enabled {
		return modules, nil
	}

	callbackFactories := make(map[string]stream.ConsumerCallbackFactory)

	for name := range settings.Subscribers {
		settings := settings.Subscribers[name]

		model := settings.SourceModel.Name
		callbackFactory := NewSubscriberCallbackFactory(transformers, outputs)

		callbackFactories[model] = callbackFactory
	}

	definer := CreateDefiner(callbackFactories)
	modules["mdlsub_subscriberapi"] = httpserver.New("mdlsub", definer)

	return modules, nil
}
