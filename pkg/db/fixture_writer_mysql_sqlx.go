package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/refl"
)

type MysqlSqlxMetaData struct {
	TableName string
}

type mysqlSqlxFixtureWriter struct {
	logger   log.Logger
	client   Client
	metadata *MysqlSqlxMetaData
}

func MysqlSqlxFixtureSetFactory[T any](metadata *MysqlSqlxMetaData, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewMysqlSqlxFixtureWriter(ctx, config, logger, metadata); err != nil {
			return nil, fmt.Errorf("failed to create mysql sqlx fixture writer for %s: %w", metadata.TableName, err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewMysqlSqlxFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, metadata *MysqlSqlxMetaData) (fixtures.FixtureWriter, error) {
	dbClient, err := ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create dbClient: %w", err)
	}

	return NewMysqlSqlxFixtureWriterWithInterfaces(logger, dbClient, metadata), nil
}

func NewMysqlSqlxFixtureWriterWithInterfaces(logger log.Logger, client Client, metadata *MysqlSqlxMetaData) fixtures.FixtureWriter {
	return &mysqlSqlxFixtureWriter{
		logger:   logger,
		client:   client,
		metadata: metadata,
	}
}

func (m *mysqlSqlxFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	for _, item := range fixtures {
		columns := refl.GetTags(item, "db")
		placeholders := funk.Map(columns, func(column string) string {
			return ":" + column
		})

		sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", m.metadata.TableName, strings.Join(columns, ","), strings.Join(placeholders, ","))

		if _, err := m.client.NamedExec(ctx, sql, item); err != nil {
			return fmt.Errorf("can not insert item %T into %s: %w", item, m.metadata.TableName, err)
		}
	}

	m.logger.Info(ctx, "loaded %d sqlx mysql fixtures", len(fixtures))

	return nil
}
