package dispatcher

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type Repository[K mdl.PossibleIdentifier, M db_repo.ModelBased[K]] struct {
	db_repo.Repository[K, M]
	dispatcher Dispatcher
	logger     log.Logger
}

func NewRepository[K mdl.PossibleIdentifier, M db_repo.ModelBased[K]](ctx context.Context, config cfg.Config, logger log.Logger, repo db_repo.Repository[K, M]) (db_repo.Repository[K, M], error) {
	disp, err := ProvideDispatcher(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not provide dispatcher: %w", err)
	}

	return &Repository[K, M]{
		Repository: repo,
		dispatcher: disp,
		logger:     logger,
	}, nil
}

func (r Repository[K, M]) Create(ctx context.Context, value M) error {
	err := r.Repository.Create(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.GetModelName(), db_repo.Create)

	err = r.dispatcher.Fire(ctx, eventName, value)
	if err != nil {
		r.logger.Error(ctx, "error on %s for event %s: %w", db_repo.Create, eventName, err)
	}

	return err
}

func (r Repository[K, M]) Update(ctx context.Context, value M) error {
	err := r.Repository.Update(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.GetModelName(), db_repo.Update)

	err = r.dispatcher.Fire(ctx, eventName, value)
	if err != nil {
		r.logger.Error(ctx, "error on %s for event %s: %w", db_repo.Update, eventName, err)
	}

	return err
}

func (r Repository[K, M]) Delete(ctx context.Context, value M) error {
	err := r.Repository.Delete(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.GetModelName(), db_repo.Delete)

	err = r.dispatcher.Fire(ctx, eventName, value)
	if err != nil {
		r.logger.Error(ctx, "error on %s for event %s: %w", db_repo.Delete, eventName, err)
	}

	return err
}
