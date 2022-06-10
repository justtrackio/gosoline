package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareLoader(fixtureSets []*FixtureSet) kernel.MiddlewareFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
		loader := NewFixtureLoader(ctx, config, logger)

		return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
			return func() {
				if err := loader.Load(ctx, fixtureSets); err != nil {
					logger.Error("can not load fixtures: %w", err)
					return
				}

				next()
			}
		}, nil
	}
}
