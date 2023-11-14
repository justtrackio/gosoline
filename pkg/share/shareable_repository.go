package share

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

type shareRepository struct {
	db_repo.Repository
	guard           guard.Guard
	logger          log.Logger
	shareRepository db_repo.Repository
}

func NewShareableRepository(ctx context.Context, config cfg.Config, logger log.Logger, repo db_repo.Repository) (*shareRepository, error) {
	guard, err := guard.NewGuard(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create guard: %w", err)
	}

	shareRepo, err := ProvideRepository(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create share repository: %w", err)
	}

	return &shareRepository{
		Repository:      repo,
		logger:          logger,
		guard:           guard,
		shareRepository: shareRepo,
	}, nil
}

func (r shareRepository) Update(ctx context.Context, value db_repo.ModelBased) error {
	entity, ok := value.(Shareable)
	if !ok {
		return fmt.Errorf("can not get entity name from given value")
	}

	err := r.Repository.Update(ctx, value)
	if err != nil {
		return err
	}

	shares, err := r.getEntityShares(ctx, entity)
	if err != nil {
		return err
	}

	for _, share := range shares {
		oldPolicy, err := r.guard.GetPolicy(ctx, share.PolicyId)
		if err != nil {
			return fmt.Errorf("can not get policy by id: %w", err)
		}

		updatedPolicy := BuildSharePolicy(share.PolicyId, entity, share.OwnerId, oldPolicy.GetActions())
		err = r.guard.UpdatePolicy(ctx, updatedPolicy)
		if err != nil {
			return fmt.Errorf("can not update policy: %w", err)
		}
	}

	return nil
}

func (r shareRepository) Delete(ctx context.Context, value db_repo.ModelBased) error {
	entity, ok := value.(Shareable)
	if !ok {
		return fmt.Errorf("can not get entity name from given value")
	}

	shares, err := r.getEntityShares(ctx, entity)
	if err != nil {
		return err
	}

	for _, share := range shares {
		err = r.shareRepository.Delete(ctx, share)
		if err != nil {
			return fmt.Errorf("can not delete share of entity: %w", err)
		}

		err = r.guard.DeletePolicy(ctx, &ladon.DefaultPolicy{ID: share.PolicyId})
		if err != nil {
			return fmt.Errorf("can not delete policy of share: %w", err)
		}
	}

	return r.Repository.Delete(ctx, value)
}

func (r shareRepository) getEntityShares(ctx context.Context, entity Shareable) ([]*Share, error) {
	qb := db_repo.NewQueryBuilder()
	qb.Where("entity_id = ? and entity_type = ?", *entity.GetId(), entity.GetEntityType())

	var result []*Share
	err := r.shareRepository.Query(ctx, qb, &result)
	if err != nil {
		return nil, fmt.Errorf("can not query shares of entity type %s: %w", entity.GetEntityType(), err)
	}

	return result, nil
}
