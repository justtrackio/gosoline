package db_repo

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/validation"
)

type validationRepository struct {
	Repository
	validator validation.Validator
}

func NewValidationRepository(validator validation.Validator, repo Repository) Repository {
	return &validationRepository{
		Repository: repo,
		validator:  validator,
	}
}

func (r validationRepository) Create(ctx context.Context, value ModelBased) error {
	err := r.validator.IsValid(ctx, value)
	if err != nil {
		return err
	}

	err = r.Repository.Create(ctx, value)

	return err
}

func (r validationRepository) Update(ctx context.Context, value ModelBased) error {
	err := r.validator.IsValid(ctx, value)
	if err != nil {
		return err
	}

	err = r.Repository.Update(ctx, value)

	return err
}
