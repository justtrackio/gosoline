package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareLoader(factory FixtureBuilderFactory) kernel.MiddlewareFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
		loader := NewFixtureLoader(ctx, config, logger)

		builder, err := factory(ctx)
		if err != nil {
			return nil, fmt.Errorf("can not build fixture loader: %w", err)
		}

		return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
			return func() {
				if err := loader.Load(ctx, builder.Fixtures()); err != nil {
					logger.Error("can not load fixtureSets: %w", err)
					return
				}

				next()
			}
		}, nil
	}
}
