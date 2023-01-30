package fixtures

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
)

type MysqlPlainFixtureValues []interface{}

type MysqlPlainMetaData struct {
	TableName string
	Columns   []string
}

type mysqlPlainFixtureWriter struct {
	logger   log.Logger
	client   db.Client
	metadata *MysqlPlainMetaData
	purger   *mysqlPurger
}

func MysqlPlainFixtureWriterFactory(metadata *MysqlPlainMetaData) FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		dbClient, err := db.NewClient(config, logger, "default")
		if err != nil {
			return nil, fmt.Errorf("can not create dbClient: %w", err)
		}

		purger, err := newMysqlPurger(config, logger, metadata.TableName)
		if err != nil {
			return nil, fmt.Errorf("can not create purger: %w", err)
		}

		return NewMysqlPlainFixtureWriterWithInterfaces(logger, dbClient, metadata, purger), nil
	}
}

func NewMysqlPlainFixtureWriterWithInterfaces(logger log.Logger, client db.Client, metadata *MysqlPlainMetaData, purger *mysqlPurger) FixtureWriter {
	return &mysqlPlainFixtureWriter{
		logger:   logger,
		client:   client,
		metadata: metadata,
		purger:   purger,
	}
}

func (m *mysqlPlainFixtureWriter) Purge(ctx context.Context) error {
	err := m.purger.purgeMysql(ctx)
	if err != nil {
		m.logger.Error("error occured during purging of table %s in plain mysql fixture loader: %w", m.metadata.TableName, err)

		return err
	}

	m.logger.Info("purged table %s for plain mysql fixtureSets", m.metadata.TableName)

	return nil
}

func (m *mysqlPlainFixtureWriter) Write(ctx context.Context, fs *FixtureSet) error {
	for _, item := range fs.Fixtures {
		fixture := item.(MysqlPlainFixtureValues)

		sql, args, err := m.buildSql(fixture)
		if err != nil {
			return err
		}

		res, err := m.client.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}

		ar, err := res.RowsAffected()
		if err != nil {
			return err
		}

		m.logger.Debug(fmt.Sprintf("affected rows while fixture loading: %d", ar))
	}

	m.logger.Info("loaded %d plain mysql fixtures", len(fs.Fixtures))

	return nil
}

func (m *mysqlPlainFixtureWriter) buildSql(values MysqlPlainFixtureValues) (string, []interface{}, error) {
	insertBuilder := squirrel.Replace(m.metadata.TableName).
		PlaceholderFormat(squirrel.Question).
		Columns(m.metadata.Columns...).
		Values(values...)

	return insertBuilder.ToSql()
}
