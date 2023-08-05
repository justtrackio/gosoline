package definitions

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

var tableMetadata = db_repo.Metadata{
	TableName:  "items",
	PrimaryKey: "id",
}

var tableHistoryMetadata = db_repo.Metadata{
	TableName: "items_histories",
}

type Item struct {
	db_repo.Model
	db_repo.ChangeHistoryEmbeddable
	Action string
	Name   string
}

type ItemsHistory struct {
	db_repo.ChangeHistoryModel
	Item
}

func NewRepository(ctx context.Context, config cfg.Config, logger log.Logger) (db_repo.Repository[uint, *Item], error) {
	settings := db_repo.Settings{
		Metadata: tableMetadata,
	}

	repository, err := db_repo.New[uint, *Item](ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("unable to create repository: %w", err)
	}

	if err := db_repo.MigrateChangeHistory(ctx, config, logger, &Item{}); err != nil {
		return nil, fmt.Errorf("unable to migrate change history: %w", err)
	}

	return repository, nil
}

func NewHistoryRepository(ctx context.Context, config cfg.Config, logger log.Logger) (db_repo.RepositoryReadOnly[uint, *ItemsHistory], error) {
	settings := db_repo.Settings{
		Metadata: tableHistoryMetadata,
	}

	repository, err := db_repo.New[uint, *ItemsHistory](ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("unable to create repository: %w", err)
	}

	return repository, nil
}
