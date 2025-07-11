//go:build !fixtures

package reslife

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func LifeCycleManagerMiddleware(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	logger = logger.WithChannel("lifecycle-manager")
	env, err := config.GetString("env")
	if err != nil {
		return nil, fmt.Errorf("failed to get env config: %w", err)
	}

	var manager *LifeCycleManager

	if manager, err = NewLifeCycleManager(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could build lifecycle manager: %w", err)
	}

	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func() {
			if env == "dev" || env == "test" {
				logger.Warn("lifecycle management is not enabled - add the 'fixtures' build tag")
			}

			if err := manager.Init(ctx); err != nil {
				logger.Error("can not handle the init lifecycle: %w", err)

				return
			}

			if err := manager.Register(ctx); err != nil {
				logger.Error("can not handle the register lifecycle: %w", err)

				return
			}

			next()
		}
	}, nil
}
