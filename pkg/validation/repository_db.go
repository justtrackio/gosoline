package validation

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type Repository[K mdl.PossibleIdentifier, M db_repo.ModelBased[K]] struct {
	db_repo.Repository[K, M]
	validator Validator
}

func NewRepository[K mdl.PossibleIdentifier, M db_repo.ModelBased[K]](validator Validator, repo db_repo.Repository[K, M]) db_repo.Repository[K, M] {
	return &Repository[K, M]{
		Repository: repo,
		validator:  validator,
	}
}

func (r Repository[K, M]) Create(ctx context.Context, value M) error {
	err := r.validator.IsValid(ctx, value)
	if err != nil {
		return err
	}

	err = r.Repository.Create(ctx, value)

	return err
}

func (r Repository[K, M]) Update(ctx context.Context, value M) error {
	err := r.validator.IsValid(ctx, value)
	if err != nil {
		return err
	}

	err = r.Repository.Update(ctx, value)

	return err
}
