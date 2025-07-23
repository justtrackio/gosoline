package db_repo

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

type mysqlOrmFixtureWriter struct {
	logger   log.Logger
	metadata *Metadata
	repo     Repository
}

func MysqlOrmFixtureSetFactory[T any](metadata *Metadata, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewMysqlOrmFixtureWriter(ctx, config, logger, metadata); err != nil {
			return nil, fmt.Errorf("failed to create mysql orm fixture writer for %s: %w", metadata.ModelId.String(), err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewMysqlOrmFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, metadata *Metadata) (fixtures.FixtureWriter, error) {
	if err := metadata.ModelId.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad model id from config: %w", err)
	}

	appId, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can not get app id from config: %w", err)
	}

	repoSettings := Settings{
		AppId:      appId,
		Metadata:   *metadata,
		ClientName: "default",
	}

	var dbSettings *db.Settings
	var repo *repository

	if dbSettings, err = db.ReadSettings(config, "default"); err != nil {
		return nil, fmt.Errorf("can not create repo: %w", err)
	}
	dbSettings.Parameters["FOREIGN_KEY_CHECKS"] = "0"

	if repo, err = NewWithDbSettings(ctx, config, logger, dbSettings, repoSettings); err != nil {
		return nil, fmt.Errorf("can not create repo: %w", err)
	}

	return NewMysqlFixtureWriterWithInterfaces(logger, metadata, repo), nil
}

func NewMysqlFixtureWriterWithInterfaces(logger log.Logger, metadata *Metadata, repo Repository) fixtures.FixtureWriter {
	return &mysqlOrmFixtureWriter{
		logger:   logger,
		metadata: metadata,
		repo:     repo,
	}
}

func (m *mysqlOrmFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	var ok bool
	var model ModelBased

	for _, item := range fixtures {
		if model, ok = item.(ModelBased); !ok {
			return fmt.Errorf("assertion failed: %T is not db_repo.ModelBased", item)
		}

		err := m.repo.Update(ctx, model)
		if err != nil {
			return err
		}
	}

	m.logger.Info(ctx, "loaded %d mysql fixtures", len(fixtures))

	return nil
}
