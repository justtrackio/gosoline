package validation

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/db-repo"
)

type Repository struct {
	db_repo.Repository
	validator Validator
}

func NewRepository(validator Validator, repo db_repo.Repository) db_repo.Repository {
	return &Repository{
		Repository: repo,
		validator:  validator,
	}
}

func (r Repository) Create(ctx context.Context, value db_repo.ModelBased) error {
	err := r.validator.IsValid(ctx, value)
	if err != nil {
		return err
	}

	err = r.Repository.Create(ctx, value)

	return err
}

func (r Repository) Update(ctx context.Context, value db_repo.ModelBased) error {
	err := r.validator.IsValid(ctx, value)
	if err != nil {
		return err
	}

	err = r.Repository.Update(ctx, value)

	return err
}
