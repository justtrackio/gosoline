package share

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/selm0/ladon"
)

type shareRepository[K mdl.PossibleIdentifier, M Shareable[K]] struct {
	db_repo.Repository[K, M]
	guard           guard.Guard
	logger          log.Logger
	shareRepository db_repo.Repository[uint, *Share]
}

func NewShareableRepository[K mdl.PossibleIdentifier, M Shareable[K]](ctx context.Context, config cfg.Config, logger log.Logger, repo db_repo.Repository[K, M]) (*shareRepository[K, M], error) {
	guard, err := guard.NewGuard(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create guard: %w", err)
	}

	shareRepo, err := ProvideRepository(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create share repository: %w", err)
	}

	return &shareRepository[K, M]{
		Repository:      repo,
		logger:          logger,
		guard:           guard,
		shareRepository: shareRepo,
	}, nil
}

func (r shareRepository[K, M]) Update(ctx context.Context, entity M) error {
	err := r.Repository.Update(ctx, entity)
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

		updatedPolicy := BuildSharePolicy[K](share.PolicyId, entity, share.OwnerId, oldPolicy.GetActions())
		err = r.guard.UpdatePolicy(ctx, updatedPolicy)
		if err != nil {
			return fmt.Errorf("can not update policy: %w", err)
		}
	}

	return nil
}

func (r shareRepository[K, M]) Delete(ctx context.Context, entity M) error {
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

	return r.Repository.Delete(ctx, entity)
}

func (r shareRepository[K, M]) getEntityShares(ctx context.Context, entity M) ([]*Share, error) {
	qb := db_repo.NewQueryBuilder()
	qb.Where("entity_id = ? and entity_type = ?", *entity.GetId(), entity.GetEntityType())

	result, err := r.shareRepository.Query(ctx, qb)
	if err != nil {
		return nil, fmt.Errorf("can not query shares of entity type %s: %w", entity.GetEntityType(), err)
	}

	return result, nil
}
