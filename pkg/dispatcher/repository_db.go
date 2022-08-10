package dispatcher

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
)

type Repository struct {
	db_repo.Repository
	logger     log.Logger
	dispatcher Dispatcher
}

func NewRepository(ctx context.Context, config cfg.Config, logger log.Logger, repo db_repo.Repository) (db_repo.Repository, error) {
	disp, err := ProvideDispatcher(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not provide dispatcher: %w", err)
	}

	return &Repository{
		Repository: repo,
		logger:     logger,
		dispatcher: disp,
	}, nil
}

func (r Repository) BatchCreate(ctx context.Context, values interface{}) error {
	err := r.Repository.BatchCreate(ctx, values)
	if err != nil {
		return err
	}

	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return fmt.Errorf("can not turn values into slice: %w", err)
	}

	eventName := fmt.Sprintf("%s.%s", r.Repository.GetModelName(), db_repo.Create)

	logger := r.logger.WithContext(ctx)

	for _, value := range valuesSlice {
		errs := r.dispatcher.Fire(ctx, eventName, value)
		for _, err := range errs {
			if err != nil {
				logger.Error("error on %s for event %s: %w", db_repo.Create, eventName, err)
			}
		}
	}

	return nil
}

func (r Repository) BatchUpdate(ctx context.Context, values interface{}) error {
	err := r.Repository.BatchUpdate(ctx, values)
	if err != nil {
		return err
	}

	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return fmt.Errorf("can not turn values into slice: %w", err)
	}

	eventName := fmt.Sprintf("%s.%s", r.Repository.GetModelName(), db_repo.Update)

	logger := r.logger.WithContext(ctx)

	for _, value := range valuesSlice {
		errs := r.dispatcher.Fire(ctx, eventName, value)
		for _, err := range errs {
			if err != nil {
				logger.Error("error on %s for event %s: %w", db_repo.Update, eventName, err)
			}
		}
	}

	return nil
}

func (r Repository) BatchDelete(ctx context.Context, values interface{}) error {
	err := r.Repository.BatchDelete(ctx, values)
	if err != nil {
		return err
	}

	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return fmt.Errorf("can not turn values into slice: %w", err)
	}

	eventName := fmt.Sprintf("%s.%s", r.Repository.GetModelName(), db_repo.Delete)

	logger := r.logger.WithContext(ctx)

	for _, value := range valuesSlice {
		errs := r.dispatcher.Fire(ctx, eventName, value)
		for _, err := range errs {
			if err != nil {
				logger.Error("error on %s for event %s: %w", db_repo.Delete, eventName, err)
			}
		}
	}

	return nil
}

func (r Repository) Create(ctx context.Context, value db_repo.ModelBased) error {
	err := r.Repository.Create(ctx, value)
	if err != nil {
		return err
	}

	eventName := fmt.Sprintf("%s.%s", r.Repository.GetModelName(), db_repo.Create)
	errs := r.dispatcher.Fire(ctx, eventName, value)

	logger := r.logger.WithContext(ctx)

	for _, err := range errs {
		if err != nil {
			logger.Error("error on %s for event %s: %w", db_repo.Create, eventName, err)
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

	logger := r.logger.WithContext(ctx)

	for _, err := range errs {
		if err != nil {
			logger.Error("error on %s for event %s: %w", db_repo.Update, eventName, err)
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

	logger := r.logger.WithContext(ctx)

	for _, err := range errs {
		if err != nil {
			logger.Error("error on %s for event %s: %w", db_repo.Delete, eventName, err)
		}
	}

	return nil
}
