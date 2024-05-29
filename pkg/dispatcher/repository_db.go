package dispatcher

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Repository struct {
	db_repo.Repository
	dispatcher Dispatcher
	logger     log.Logger
}

func NewRepository(ctx context.Context, config cfg.Config, logger log.Logger, repo db_repo.Repository) (db_repo.Repository, error) {
	factory, err := NewRepositoryFactory(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	return factory(repo), nil
}

func NewRepositoryFactory(ctx context.Context, config cfg.Config, logger log.Logger) (func(db_repo.Repository) db_repo.Repository, error) {
	disp, err := ProvideDispatcher(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not provide dispatcher: %w", err)
	}

	return func(repo db_repo.Repository) db_repo.Repository {
		return &Repository{
			Repository: repo,
			dispatcher: disp,
			logger:     logger,
		}
	}, nil
}

func (r Repository) Create(ctx context.Context, value db_repo.ModelBased) error {
	err := r.Repository.Create(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.Repository.GetModelName(), db_repo.Create)
	errs := r.dispatcher.Fire(ctx, eventName, value)

	for _, err := range errs {
		if err != nil {
			r.logger.Error("error on %s for event %s: %w", db_repo.Create, eventName, err)
		}
	}

	return nil
}

func (r Repository) Update(ctx context.Context, value db_repo.ModelBased) error {
	err := r.Repository.Update(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.Repository.GetModelName(), db_repo.Update)
	errs := r.dispatcher.Fire(ctx, eventName, value)

	for _, err := range errs {
		if err != nil {
			r.logger.Error("error on %s for event %s: %w", db_repo.Update, eventName, err)
		}
	}

	return nil
}

func (r Repository) Delete(ctx context.Context, value db_repo.ModelBased) error {
	err := r.Repository.Delete(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.Repository.GetModelName(), db_repo.Delete)
	errs := r.dispatcher.Fire(ctx, eventName, value)

	for _, err := range errs {
		if err != nil {
			r.logger.Error("error on %s for event %s: %w", db_repo.Delete, eventName, err)
		}
	}

	return nil
}
