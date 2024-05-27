package validation

import (
	"context"

	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type OperationValidatingRepository[K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	dbRepo.Repository[K, M]
	validator Validator
}

func NewOperationValidatingRepository[K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](validator Validator, repo dbRepo.Repository[K, M]) dbRepo.Repository[K, M] {
	return &OperationValidatingRepository[K, M]{
		Repository: repo,
		validator:  validator,
	}
}

func (r OperationValidatingRepository[K, M]) Create(ctx context.Context, value M) error {
	err := r.validator.IsValid(ctx, value, dbRepo.Create)
	if err != nil {
		return err
	}

	return r.Repository.Create(ctx, value)
}

func (r OperationValidatingRepository[K, M]) Update(ctx context.Context, value M) error {
	err := r.validator.IsValid(ctx, value, dbRepo.Update)
	if err != nil {
		return err
	}

	return r.Repository.Update(ctx, value)
}

func (r OperationValidatingRepository[K, M]) Delete(ctx context.Context, value M) error {
	err := r.validator.IsValid(ctx, value, dbRepo.Delete)
	if err != nil {
		return err
	}

	return r.Repository.Delete(ctx, value)
}
