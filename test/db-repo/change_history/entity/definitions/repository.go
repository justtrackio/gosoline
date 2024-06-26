package definitions

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

var tableMetadata = db_repo.Metadata{
	TableName: "items",
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

func NewRepository(ctx context.Context, config cfg.Config, logger log.Logger) (db_repo.Repository, error) {
	settings := db_repo.Settings{
		Metadata: tableMetadata,
	}

	repository, err := db_repo.New(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("unable to create repository: %w", err)
	}

	manager, err := db_repo.NewChangeHistoryManager(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("unable to create change history manager: %w", err)
	}

	err = manager.RunMigration(&Item{})
	if err != nil {
		return nil, fmt.Errorf("failed to run history migration: %w", err)
	}

	return repository, nil
}

func NewHistoryRepository(ctx context.Context, config cfg.Config, logger log.Logger) (db_repo.RepositoryReadOnly, error) {
	settings := db_repo.Settings{
		Metadata: tableHistoryMetadata,
	}

	repository, err := db_repo.New(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("unable to create repository: %w", err)
	}

	return repository, nil
}
