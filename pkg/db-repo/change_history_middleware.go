package db_repo

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareChangeHistory(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Middleware, error) {
	var err error
	var manager *ChangeHistoryManager

	if manager, err = ProvideChangeHistoryManager(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("can not access the change history manager: %w", err)
	}

	return func(next kernel.MiddlewareHandler) kernel.MiddlewareHandler {
		return func() {
			if err = manager.RunMigrations(); err != nil {
				logger.Error("can not run change history migrations: %w", err)

				return
			}

			next()
		}
	}, nil
}
