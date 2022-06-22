package validation

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/refl"
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

func (r Repository) BatchCreate(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return err
	}

	for _, value := range valuesSlice {
		if err := r.validator.IsValid(ctx, value); err != nil {
			return err
		}
	}

	return r.Repository.BatchCreate(ctx, values)
}

func (r Repository) BatchUpdate(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return err
	}

	for _, value := range valuesSlice {
		if err := r.validator.IsValid(ctx, value); err != nil {
			return err
		}
	}

	return r.Repository.BatchUpdate(ctx, values)
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
