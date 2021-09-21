package validation

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/db-repo"
)

type OperationValidatingRepository struct {
	db_repo.Repository
	validator Validator
}

func NewOperationValidatingRepository(validator Validator, repo db_repo.Repository) db_repo.Repository {
	return &OperationValidatingRepository{
		Repository: repo,
		validator:  validator,
	}
}

func (r OperationValidatingRepository) Create(ctx context.Context, value db_repo.ModelBased) error {
	err := r.validator.IsValid(ctx, value, db_repo.Create)
	if err != nil {
		return err
	}

	return r.Repository.Create(ctx, value)
}

func (r OperationValidatingRepository) Update(ctx context.Context, value db_repo.ModelBased) error {
	err := r.validator.IsValid(ctx, value, db_repo.Update)
	if err != nil {
		return err
	}

	return r.Repository.Update(ctx, value)
}

func (r OperationValidatingRepository) Delete(ctx context.Context, value db_repo.ModelBased) error {
	err := r.validator.IsValid(ctx, value, db_repo.Delete)
	if err != nil {
		return err
	}

	return r.Repository.Delete(ctx, value)
}
