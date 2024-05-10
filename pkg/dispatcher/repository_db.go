package dispatcher

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Repository struct {
	db_repo.Repository
	dispatcher Dispatcher
	logger     log.Logger
}

func NewRepository(ctx context.Context, config cfg.Config, logger log.Logger, repo db_repo.Repository) (db_repo.Repository, error) {
	disp, err := ProvideDispatcher(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not provide dispatcher: %w", err)
	}

	return &Repository{
		Repository: repo,
		dispatcher: disp,
		logger:     logger,
	}, nil
}

func (r Repository) Create(ctx context.Context, value db_repo.ModelBased) error {
	err := r.Repository.Create(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.GetModelName(), db_repo.Create)

	err = r.dispatcher.Fire(ctx, eventName, value)
	if err != nil {
		r.logger.WithContext(ctx).Error("error on %s for event %s: %w", db_repo.Create, eventName, err)
	}

	return err
}

func (r Repository) Update(ctx context.Context, value db_repo.ModelBased) error {
	err := r.Repository.Update(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.GetModelName(), db_repo.Update)

	err = r.dispatcher.Fire(ctx, eventName, value)
	if err != nil {
		r.logger.WithContext(ctx).Error("error on %s for event %s: %w", db_repo.Update, eventName, err)
	}

	return err
}

func (r Repository) Delete(ctx context.Context, value db_repo.ModelBased) error {
	err := r.Repository.Delete(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.GetModelName(), db_repo.Delete)

	err = r.dispatcher.Fire(ctx, eventName, value)
	if err != nil {
		r.logger.Error("error on %s for event %s: %w", db_repo.Delete, eventName, err)
	}

	return err
}
