package fixtures

import (
	"context"
	"database/sql"
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
	logger       log.Logger
	client       db.Client
	metadata     *MysqlPlainMetaData
	purger       *mysqlPurger
	stateService *mysqlStateService
}

func MysqlPlainFixtureWriterFactory(metadata *MysqlPlainMetaData) FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		dbClient, err := db.ProvideClient(ctx, config, logger, "default")
		if err != nil {
			return nil, fmt.Errorf("can not create dbClient: %w", err)
		}

		purger, err := newMysqlPurger(ctx, config, logger, metadata.TableName)
		if err != nil {
			return nil, fmt.Errorf("can not create purger: %w", err)
		}

		state, err := provideMysqlStateService(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create state serveice: %w", err)
		}

		return NewMysqlPlainFixtureWriterWithInterfaces(logger, dbClient, metadata, purger, state), nil
	}
}

func NewMysqlPlainFixtureWriterWithInterfaces(logger log.Logger, client db.Client, metadata *MysqlPlainMetaData, purger *mysqlPurger, state *mysqlStateService) FixtureWriter {
	return &mysqlPlainFixtureWriter{
		logger:       logger,
		client:       client,
		metadata:     metadata,
		purger:       purger,
		stateService: state,
	}
}

func (m *mysqlPlainFixtureWriter) Purge(ctx context.Context) error {
	err := m.purger.purgeMysql(ctx)
	if err != nil {
		m.logger.Error("error occurred during purging of table %s in plain mysql fixture loader: %w", m.metadata.TableName, err)

		return fmt.Errorf("failed to purge mysql table %s: %w", m.metadata.TableName, err)
	}

	m.logger.Info("purged table %s for plain mysql fixtureSets", m.metadata.TableName)

	return nil
}

func (m *mysqlPlainFixtureWriter) buildSql(fixtures []any) ([]string, [][]any, error) {
	cols := make([]string, len(m.metadata.Columns))
	for i, col := range m.metadata.Columns {
		cols[i] = fmt.Sprintf("`%s`", col) // need to quote these, as they might contain mysql keywords
	}

	stmts := make([]string, 0)
	argss := make([][]any, 0)

	insertBuilder := squirrel.
		Replace(m.metadata.TableName).
		PlaceholderFormat(squirrel.Question).
		Columns(cols...)
	offset := 0
	for i, values := range fixtures {
		// if we exceed the max amount of parameters for a prepared statement, make it a new statement instead
		if ((i+1)*len(cols))-offset > 65536 {
			offset = i * len(cols)
			stmt, args, err := insertBuilder.ToSql()
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

	stmt, args, err := insertBuilder.ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build sql statement: %w", err)
	}

	stmts = append(stmts, stmt)
	argss = append(argss, args)

	return stmts, argss, nil
}

func (m *mysqlPlainFixtureWriter) Write(ctx context.Context, fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	stmts, argss, err := m.buildSql(fs.Fixtures)
	if err != nil {
		return fmt.Errorf("failed to build sqlers for fixture loading: %w", err)
	}

	var ress []sql.Result
	var sqls []db.Sqler

	sqls = append(sqls, db.SqlFmt(foreignKeyChecksStatement, nil, 0))
	for i, stmt := range stmts {
		sqls = append(sqls, db.SqlFmt(stmt, nil, argss[i]...))
	}
	sqls = append(sqls, db.SqlFmt(foreignKeyChecksStatement, nil, 1))

	ress, err = m.client.ExecMultiInTx(ctx, sqls...)
	if err != nil {
		m.logger.Error("error writing fixtures: %w", err)

		return fmt.Errorf("failed to execute fixture loading queries in transaction: %w", err)
	}

	if len(ress) < len(sqls) {
		return fmt.Errorf("expected %d results, got %d", len(sqls), len(ress))
	}

	m.logger.Info("loaded %d %s plain mysql fixtures", len(fs.Fixtures), fs.FixtureSetName)

	if len(fs.FixtureSetName) == 0 {
		return nil
	}

	_, err = m.stateService.Persist(ctx, fs.FixtureSetName, m.metadata.TableName)
	if err != nil {
		return fmt.Errorf("failed to persist fixture state: %w", err)
	}

	return nil
}
