package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

type mysqlOrmFixtureWriter struct {
	logger   log.Logger
	metadata *db_repo.Metadata
	repo     db_repo.Repository
	purger   *mysqlPurger
}

func NewMysqlOrmFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, metadata *db_repo.Metadata) (FixtureWriter, error) {
	metadata.ModelId.PadFromConfig(config)

	settings := db_repo.Settings{
		AppId:    cfg.GetAppIdFromConfig(config),
		Metadata: *metadata,
	}

	repo, err := db_repo.New(ctx, config, logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not create repo: %w", err)
	}

	purger, err := newMysqlPurger(ctx, config, logger, metadata.TableName)
	if err != nil {
		return nil, fmt.Errorf("can not create purger: %w", err)
	}

	return NewMysqlFixtureWriterWithInterfaces(logger, metadata, repo, purger), nil
}

func NewMysqlFixtureWriterWithInterfaces(logger log.Logger, metadata *db_repo.Metadata, repo db_repo.Repository, purger *mysqlPurger) FixtureWriter {
	return &mysqlOrmFixtureWriter{
		logger:   logger,
		metadata: metadata,
		repo:     repo,
		purger:   purger,
	}
}

func (m *mysqlOrmFixtureWriter) Purge(ctx context.Context) error {
	err := m.purger.purgeMysql(ctx)
	if err != nil {
		m.logger.Error("error occured during purging of table %s in plain mysql fixture loader: %w", m.metadata.TableName, err)

		return err
	}

	m.logger.Info("purged table for orm mysql fixtureSets")

	return nil
}

func (m *mysqlOrmFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	for _, item := range fixtures {
		model := item.(db_repo.ModelBased)

		err := m.repo.Update(ctx, model)
		if err != nil {
			return err
		}
	}

	m.logger.Info("loaded %d mysql fixtures", len(fixtures))

	return nil
}
