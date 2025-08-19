//go:build fixtures

package reslife

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func LifeCycleManagerMiddleware(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	logger = logger.WithChannel("lifecycle-manager")

	var err error
	var manager *LifeCycleManager
	var container *fixtures.Container
	var loader fixtures.FixtureLoader

	if manager, err = NewLifeCycleManager(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could build lifecycle manager: %w", err)
	}

	if container, err = fixtures.ProvideContainer(ctx); err != nil {
		return nil, fmt.Errorf("could not load fixture container: %w", err)
	}

	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func(ctx context.Context) {
			if loader, err = container.Build(ctx, config, logger); err != nil {
				logger.Error("can not build fixture loader: %w", err)

				return
			}

			if err := manager.Create(ctx); err != nil {
				logger.Error("can not handle the create lifecycle: %w", err)

				return
			}

			if err := manager.Init(ctx); err != nil {
				logger.Error("can not handle the init lifecycle: %w", err)

				return
			}

			if err := manager.Register(ctx); err != nil {
				logger.Error("can not handle the register lifecycle: %w", err)

				return
			}

			if err := manager.Purge(ctx); err != nil {
				logger.Error("can not handle the purge lifecycle: %w", err)

				return
			}

			if err := loader.Load(ctx); err != nil {
				logger.Error("can not load fixtures: %w", err)

				return
			}

			next(ctx)
		}
	}, nil
}
