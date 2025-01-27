package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareLoader(group string, factory FixtureSetsFactory, postProcessorFactories ...PostProcessorFactory) kernel.MiddlewareFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
		settings := unmarshalFixtureLoaderSettings(config)
		logger = logger.WithChannel("fixtures").WithFields(map[string]any{
			"group": group,
		})

		if !settings.Enabled {
			return disabledMiddleware(logger)
		}

		var err error
		var loader FixtureLoader
		var fixtureSets []FixtureSet

		if loader, err = NewFixtureLoader(ctx, config, logger, postProcessorFactories...); err != nil {
			return nil, fmt.Errorf("could not create fixture loader: %w", err)
		}

		if fixtureSets, err = factory(ctx, config, logger, group); err != nil {
			return nil, fmt.Errorf("can not build fixture sets: %w", err)
		}

		return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
			return func() {
				if err = loader.Load(ctx, group, fixtureSets); err != nil {
					logger.Error("can not load fixture sets: %w", err)

					return
				}

				next()
			}
		}, nil
	}
}

func disabledMiddleware(logger log.Logger) (kernel.Middleware, error) {
	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func() {
			logger.Info("fixture loader is not enabled")
			next()
		}
	}, nil
}
