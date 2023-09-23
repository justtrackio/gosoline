package share

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareShares(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	var err error

	manager, err := ProvideShareManager(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create share manager: %w", err)
	}

	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func() {
			if err := manager.SetupShareTable(); err != nil {
				logger.Error("can not setup share tables: %s", err.Error())

				return
			}

			next()
		}
	}, nil
}
