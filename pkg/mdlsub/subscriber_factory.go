package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func NewSubscriberFactory(transformerFactoryMap TransformerMapTypeVersionFactories) kernel.ModuleMultiFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		return SubscriberFactory(ctx, config, logger, transformerFactoryMap)
	}
}

func SubscriberFactory(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	transformerFactories TransformerMapTypeVersionFactories,
) (map[string]kernel.ModuleFactory, error) {
	var err error
	var core SubscriberCore
	var settings *Settings

	if settings, err = unmarshalSettings(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscriber settings: %w", err)
	}

	if core, err = NewSubscriberCore(ctx, config, logger, settings.Subscribers, transformerFactories); err != nil {
		return nil, fmt.Errorf("failed to create subscriber core: %w", err)
	}

	modules := make(map[string]kernel.ModuleFactory)

	for name, subscriberSettings := range settings.Subscribers {
		subscriberFQN := GetSubscriberFQN(name, subscriberSettings.SourceModel)

		if _, ok := modules[subscriberFQN]; ok {
			continue
		}

		callbackFactory := NewSubscriberCallbackFactory(core, subscriberSettings.SourceModel, subscriberSettings.PersistGraceTime)
		modules[subscriberFQN] = stream.NewUntypedConsumer(subscriberFQN, callbackFactory)
	}

	return modules, nil
}
