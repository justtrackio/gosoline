package condition

import (
	"context"
	"github.com/applike/gosoline/pkg/db-repo"
)

type Repository struct {
	db_repo.Repository
	conditions Conditions
}

func NewRepository(conditions Conditions, repo db_repo.Repository) db_repo.Repository {
	return &Repository{
		Repository: repo,
		conditions: conditions,
	}
}

func (r Repository) Create(ctx context.Context, value db_repo.ModelBased) error {
	err := r.conditions.IsValid(ctx, value, db_repo.Create)

	if err != nil {
		return err
	}

	return r.Repository.Create(ctx, value)
}

func (r Repository) Update(ctx context.Context, value db_repo.ModelBased) error {
	err := r.conditions.IsValid(ctx, value, db_repo.Update)

	if err != nil {
		return err
	}

	return r.Repository.Update(ctx, value)
}

func (r Repository) Delete(ctx context.Context, value db_repo.ModelBased) error {
	err := r.conditions.IsValid(ctx, value, db_repo.Delete)

	if err != nil {
		return err
	}

	return r.Repository.Delete(ctx, value)
}
