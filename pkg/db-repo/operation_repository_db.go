package db_repo

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/validation"
)

type OperationValidatingRepository struct {
	Repository
	validator validation.Validator
}

func NewOperationValidatingRepository(validator validation.Validator, repo Repository) Repository {
	return &OperationValidatingRepository{
		Repository: repo,
		validator:  validator,
	}
}

func (r OperationValidatingRepository) Create(ctx context.Context, value ModelBased) error {
	err := r.validator.IsValid(ctx, value, Create)
	if err != nil {
		return err
	}

	return r.Repository.Create(ctx, value)
}

func (r OperationValidatingRepository) Update(ctx context.Context, value ModelBased) error {
	err := r.validator.IsValid(ctx, value, Update)
	if err != nil {
		return err
	}

	return r.Repository.Update(ctx, value)
}

func (r OperationValidatingRepository) Delete(ctx context.Context, value ModelBased) error {
	err := r.validator.IsValid(ctx, value, Delete)
	if err != nil {
		return err
	}

	return r.Repository.Delete(ctx, value)
}
