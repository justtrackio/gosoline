package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

type MysqlPlainFixtureValues []any

type MysqlPlainMetaData struct {
	TableName string
	Columns   []string
}

type mysqlPlainFixtureWriter struct {
	logger   log.Logger
	client   Client
	metadata *MysqlPlainMetaData
}

func MysqlPlainFixtureSetFactory[T any](metadata *MysqlPlainMetaData, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewMysqlPlainFixtureWriter(ctx, config, logger, metadata); err != nil {
			return nil, fmt.Errorf("failed to create mysql plain fixture writer for %s: %w", metadata.TableName, err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewMysqlPlainFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, metadata *MysqlPlainMetaData) (fixtures.FixtureWriter, error) {
	dbClient, err := ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create dbClient: %w", err)
	}

	return NewMysqlPlainFixtureWriterWithInterfaces(logger, dbClient, metadata), nil
}

func NewMysqlPlainFixtureWriterWithInterfaces(logger log.Logger, client Client, metadata *MysqlPlainMetaData) fixtures.FixtureWriter {
	return &mysqlPlainFixtureWriter{
		logger:   logger,
		client:   client,
		metadata: metadata,
	}
}

func (m *mysqlPlainFixtureWriter) buildSql(fixtures []any) (stmts []string, argss [][]any, err error) {
	cols := make([]string, len(m.metadata.Columns))
	for i, col := range m.metadata.Columns {
		cols[i] = fmt.Sprintf("`%s`", col) // need to quote these, as they might contain mysql keywords
	}

	insertBuilder := squirrel.
		Replace(m.metadata.TableName).
		PlaceholderFormat(squirrel.Question).
		Columns(cols...)
	offset := 0
	var stmt string
	var args []any

	for i, values := range fixtures {
		// if we exceed the max amount of parameters for a prepared statement, make it a new statement instead
		if ((i+1)*len(cols))-offset > maxMysqlPreparedStatementArgs {
			offset = i * len(cols)

			stmt, args, err = insertBuilder.ToSql()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to build sql statement: %w", err)
			}

			stmts = append(stmts, stmt)
			argss = append(argss, args)
			insertBuilder = squirrel.
				Replace(m.metadata.TableName).
				PlaceholderFormat(squirrel.Question).
				Columns(cols...)
		}

		pVals, ok := values.(MysqlPlainFixtureValues)
		if !ok {
			return nil, nil, fmt.Errorf("mysqlPlainFixtureWriter values for table %s are type %T, but should be fixtures.MysqlPlainFixtureValues", m.metadata.TableName, values)
		}

		insertBuilder = insertBuilder.Values(pVals...)
	}

	stmt, args, err = insertBuilder.ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build sql statement: %w", err)
	}

	stmts = append(stmts, stmt)
	argss = append(argss, args)

	return stmts, argss, nil
}

func (m *mysqlPlainFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	stmts, argss, err := m.buildSql(fixtures)
	if err != nil {
		return fmt.Errorf("failed to build sqlers for fixture loading: %w", err)
	}

	var ress []sql.Result
	var sqls []Sqler

	sqls = append(sqls, SqlFmt("SET FOREIGN_KEY_CHECKS = ?", nil, 0))
	for i, stmt := range stmts {
		sqls = append(sqls, SqlFmt(stmt, nil, argss[i]...))
	}
	sqls = append(sqls, SqlFmt("SET FOREIGN_KEY_CHECKS = ?", nil, 1))

	ress, err = m.client.ExecMultiInTx(ctx, sqls...)
	if err != nil {
		m.logger.Error(ctx, "error writing fixtures: %w", err)

		return fmt.Errorf("failed to execute fixture loading queries in transaction: %w", err)
	}

	if len(ress) < len(sqls) {
		return fmt.Errorf("expected %d results, got %d", len(sqls), len(ress))
	}

	m.logger.Info(ctx, "loaded %d plain mysql fixtures", len(fixtures))

	return nil
}
