package fixtures

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
)

type MysqlSqlxMetaData struct {
	TableName string
}

type mysqlSqlxFixtureWriter struct {
	logger   log.Logger
	client   db.Client
	metadata *MysqlSqlxMetaData
	purger   *mysqlPurger
}

func MysqlSqlxFixtureWriterFactory(metadata *MysqlSqlxMetaData) FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		dbClient, err := db.ProvideClient(ctx, config, logger, "default")
		if err != nil {
			return nil, fmt.Errorf("can not create dbClient: %w", err)
		}

		purger, err := newMysqlPurger(ctx, config, logger, metadata.TableName)
		if err != nil {
			return nil, fmt.Errorf("can not create purger: %w", err)
		}

		return NewMysqlSqlxFixtureWriterWithInterfaces(logger, dbClient, metadata, purger), nil
	}
}

func NewMysqlSqlxFixtureWriterWithInterfaces(logger log.Logger, client db.Client, metadata *MysqlSqlxMetaData, purger *mysqlPurger) FixtureWriter {
	return &mysqlSqlxFixtureWriter{
		logger:   logger,
		client:   client,
		metadata: metadata,
		purger:   purger,
	}
}

func (m *mysqlSqlxFixtureWriter) Purge(ctx context.Context) error {
	err := m.purger.purgeMysql(ctx)
	if err != nil {
		m.logger.Error("error occurred during purging of table %s in plain mysql fixture loader: %w", m.metadata.TableName, err)

		return err
	}

	m.logger.Info("purged table %s for plain mysql fixtureSets", m.metadata.TableName)

	return nil
}

func (m *mysqlSqlxFixtureWriter) Write(ctx context.Context, fs *FixtureSet) error {
	for _, item := range fs.Fixtures {
		columns := refl.GetTags(item, "db")
		placeholders := funk.Map(columns, func(column string) string {
			return ":" + column
		})

		sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", m.metadata.TableName, strings.Join(columns, ","), strings.Join(placeholders, ","))

		if _, err := m.client.NamedExec(ctx, sql, item); err != nil {
			return fmt.Errorf("can not insert item %T into %s: %w", item, m.metadata.TableName, err)
		}
	}

	m.logger.Info("loaded %d sqlx mysql fixtures", len(fs.Fixtures))

	return nil
}
