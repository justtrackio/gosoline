package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareLoader(group string, factory FixtureSetsFactory) kernel.MiddlewareFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
		var err error
		var fixtureSets []FixtureSet

		logger = logger.WithChannel("fixtures").WithFields(map[string]any{
			"group": group,
		})

		loader := NewFixtureLoader(ctx, config, logger)

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
