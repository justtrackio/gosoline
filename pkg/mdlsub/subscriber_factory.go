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

func SubscriberFactory(ctx context.Context, config cfg.Config, logger log.Logger, transformerFactories TransformerMapTypeVersionFactories) (map[string]kernel.ModuleFactory, error) {
	var err error
	var core SubscriberCore
	var settings *Settings
	var subscriberFQN string

	if settings, err = unmarshalSettings(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscriber settings: %w", err)
	}

	if core, err = NewSubscriberCore(ctx, config, logger, settings.Subscribers, transformerFactories); err != nil {
		return nil, fmt.Errorf("failed to create subscriber core: %w", err)
	}

	modules := make(map[string]kernel.ModuleFactory)

	for name, subscriberSettings := range settings.Subscribers {
		if subscriberFQN, err = GetSubscriberFQN(config, name, subscriberSettings.SourceModel); err != nil {
			return nil, fmt.Errorf("can not get subscriber fqn for subscriber %s: %w", name, err)
		}

		if _, ok := modules[subscriberFQN]; ok {
			if subscriberSettings.SourceModel.Shared && subscriberSettings.TargetModel.Shared {
				continue
			}

			// if two subscribers result in the same FQN, then they must both be shared. as this is not the case, we have a misconfigured app
			// and should report this as an error (otherwise, we would just subscribe to part of the data we need to)
			return nil, fmt.Errorf("duplicate subscriber name %q for source model %q", subscriberFQN, subscriberSettings.SourceModel)
		}

		callbackFactory := NewSubscriberCallbackFactory(core, subscriberSettings.SourceModel, subscriberSettings.PersistGraceTime)
		modules[subscriberFQN] = stream.NewUntypedConsumer(subscriberFQN, callbackFactory)
	}

	return modules, nil
}
