package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareLoader(factory FixtureSetsFactory) kernel.MiddlewareFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
		logger = logger.WithChannel("fixtures")
		loader := NewFixtureLoader(ctx, config, logger)

		fixtureSets, err := factory(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not build fixture sets: %w", err)
		}

		return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
			return func() {
				logger.Info("loading fixtures")
				start := time.Now()

				if err = loader.Load(ctx, fixtureSets); err != nil {
					logger.Error("can not load fixture sets: %w", err)
					return
				}

				logger.Info("done loading fixtures in %s", time.Since(start))

				next()
			}
		}, nil
	}
}
