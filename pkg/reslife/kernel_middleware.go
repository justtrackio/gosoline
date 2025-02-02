package reslife

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelLifeCycleManager(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	logger = logger.WithChannel("lifecycle-manager")

	var err error
	var manager *LifeCycleManager

	if manager, err = ProvideLifeCycleManager(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could build lifecycle manager: %w", err)
	}

	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func() {
			if err := manager.Create(ctx); err != nil {
				logger.Error("can not handle the create lifecycle: %w", err)

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

			next()
		}
	}, nil
}
