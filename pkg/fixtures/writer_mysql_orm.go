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

func MysqlOrmFixtureSetFactory[T any](metadata *db_repo.Metadata, data NamedFixtures[T], options ...FixtureSetOption) FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureSet, error) {
		var err error
		var writer FixtureWriter

		if writer, err = NewMysqlOrmFixtureWriter(ctx, config, logger, metadata); err != nil {
			return nil, fmt.Errorf("failed to create mysql orm fixture writer for %s: %w", metadata.ModelId.String(), err)
		}

		return NewSimpleFixtureSet(data, writer, options...), nil
	}
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

	purger, err := NewMysqlPurger(ctx, config, logger, metadata.TableName)
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
	err := m.purger.Purge(ctx)
	if err != nil {
		m.logger.Error("error occured during purging of table %s in plain mysql fixture loader: %w", m.metadata.TableName, err)

		return err
	}

	m.logger.Info("purged table for orm mysql fixtureSets")

	return nil
}

func (m *mysqlOrmFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	var ok bool
	var model db_repo.ModelBased

	for _, item := range fixtures {
		if model, ok = item.(db_repo.ModelBased); !ok {
			return fmt.Errorf("can not convert model %T to db_repo.ModelBased", item)
		}

		err := m.repo.Update(ctx, model)
		if err != nil {
			return err
		}
	}

	m.logger.Info("loaded %d mysql fixtures", len(fixtures))

	return nil
}
