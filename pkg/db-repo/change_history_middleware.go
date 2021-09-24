package db_repo

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func KernelMiddlewareChangeHistory(ctx context.Context, config cfg.Config, logger log.Logger, next kernel.Handler) kernel.Handler {
	return func() {
		var err error
		var manager *ChangeHistoryManager

		if manager, err = ProvideChangeHistoryManager(ctx, config, logger); err != nil {
			logger.Error("can not access the change history manager: %w", err)
			return
		}

		if err = manager.RunMigrations(); err != nil {
			logger.Error("can not run change history migrations: %w", err)
			return
		}

		next()
	}
}
