package share

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/guard"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/selm0/ladon"
)

type WithPolicy interface {
	GetPolicyId() string
}

type repositoryCtxKey string

type sqlRepository struct {
	db_repo.Repository
	guard guard.Guard
}

func ProvideRepository(ctx context.Context, config cfg.Config, logger log.Logger) (db_repo.Repository, error) {
	return appctx.Provide(ctx, repositoryCtxKey("ShareRepository"), func() (db_repo.Repository, error) {
		return newRepository(ctx, config, logger)
	})
}

func newRepository(ctx context.Context, config cfg.Config, logger log.Logger) (db_repo.Repository, error) {
	var settings Settings
	if err := config.UnmarshalKey("shares", &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal share repository settings: %w", err)
	}
	tn := settings.TableName

	fieldMapping := func(field string) db_repo.FieldMapping {
		return db_repo.NewFieldMapping(fmt.Sprintf("%s.%s", tn.Share, field))
	}

	dbSettings := db_repo.Settings{
		Metadata: db_repo.Metadata{
			TableName:  tn.Share,
			PrimaryKey: fmt.Sprintf("%s.id", tn.Share),
			Mappings: db_repo.FieldMappings{
				"shares.entityId":   fieldMapping("entity_id"),
				"shares.entityType": fieldMapping("entity_type"),
				"shares.ownerId":    fieldMapping("owner_id"),
				"shares.policyId":   fieldMapping("policy_id"),
			},
		},
	}

	repo, err := db_repo.New(ctx, config, logger, dbSettings)
	if err != nil {
		return nil, fmt.Errorf("can not create repository: %w", err)
	}

	guard, err := guard.NewGuard(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create guard: %w", err)
	}

	return &sqlRepository{
		Repository: repo,
		guard:      guard,
	}, nil
}

func (r sqlRepository) Delete(ctx context.Context, value db_repo.ModelBased) error {
	s, ok := value.(WithPolicy)
	if !ok {
		return fmt.Errorf("can not get policy id from given entity")
	}

	err := r.Repository.Delete(ctx, value)
	if err != nil {
		return err
	}

	err = r.guard.DeletePolicy(ctx, &ladon.DefaultPolicy{ID: s.GetPolicyId()})
	if err != nil {
		return err
	}

	return nil
}
