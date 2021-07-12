package validation

import (
	"context"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/mon"
)

type Repository struct {
	logger mon.Logger
	db_repo.Repository
	validator Validator
}

func NewRepository(logger mon.Logger, validator Validator, repo db_repo.Repository) db_repo.Repository {
	return &Repository{
		logger: logger,
		Repository: repo,
		validator:  validator,
	}
}

func (r Repository) Create(ctx context.Context, value db_repo.ModelBased) error {
	logger := r.logger.WithContext(ctx)

	err := r.validator.IsValid(ctx, value)

	if err != nil {
		logger.Warn("validation failed: %w", err)
		return err
	}

	err = r.Repository.Create(ctx, value)

	return err
}

func (r Repository) Update(ctx context.Context, value db_repo.ModelBased) error {
	logger := r.logger.WithContext(ctx)

	err := r.validator.IsValid(ctx, value)

	if err != nil {
		logger.Warn("validation failed: %w", err)
		return err
	}

	err = r.Repository.Update(ctx, value)

	return err
}
