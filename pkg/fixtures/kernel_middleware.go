package fixtures

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/fixtures/writers"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareLoader(fixtureSets []*writers.FixtureSet) kernel.Middleware {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
		return func() {
			loader := NewFixtureLoader(ctx, config, logger)

			if err := loader.Load(ctx, fixtureSets); err != nil {
				logger.Error("can not load fixtures: %w", err)
				return
			}

			next()
		}
	}
}
