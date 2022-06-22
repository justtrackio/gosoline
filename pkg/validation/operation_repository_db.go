package validation

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/refl"
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

func (r OperationValidatingRepository) BatchCreate(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return err
	}

	for _, value := range valuesSlice {
		if err = r.validator.IsValid(ctx, value, db_repo.Create); err != nil {
			return err
		}
	}

	return r.Repository.BatchCreate(ctx, values)
}

func (r OperationValidatingRepository) BatchUpdate(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return err
	}

	for _, value := range valuesSlice {
		if err = r.validator.IsValid(ctx, value, db_repo.Update); err != nil {
			return err
		}
	}

	return r.Repository.BatchUpdate(ctx, values)
}

func (r OperationValidatingRepository) BatchDelete(ctx context.Context, values interface{}) error {
	valuesSlice, err := refl.InterfaceToInterfaceSlice(values)
	if err != nil {
		return err
	}

	for _, value := range valuesSlice {
		if err = r.validator.IsValid(ctx, value, db_repo.Delete); err != nil {
			return err
		}
	}

	return r.Repository.BatchDelete(ctx, values)
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
