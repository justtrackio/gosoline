package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	maxMysqlPreparedStatementArgs = 65536
)

type DataImporter interface {
	ImportDatabase(ctx context.Context, database string, data DatabaseData) (int, error)
}

type dataImporter struct {
	logger        log.Logger
	settings      *Settings
	clientFactory func(database string) (Client, error)
}

func NewDataImporter(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*dataImporter, error) {
	var err error
	var settings *Settings

	if settings, err = readSettings(config, name); err != nil {
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	clientFactory := func(database string) (Client, error) {
		dbSettings := *settings
		dbSettings.Uri.Database = database

		dbSettings.Parameters = map[string]string{
			"FOREIGN_KEY_CHECKS": "0",
		}
		for k, v := range settings.Parameters {
			dbSettings.Parameters[k] = v
		}

		return NewClientWithSettings(ctx, config, logger, name, &dbSettings)
	}

	return &dataImporter{
		logger:        logger,
		settings:      settings,
		clientFactory: clientFactory,
	}, nil
}

func (i *dataImporter) ImportDatabase(ctx context.Context, database string, data DatabaseData) (int, error) {
	var err error
	var dbClient Client
	var res *Result
	var ress []sql.Result
	var sqls []Sqler
	var stmts, tables []string
	var argss [][]any
	var fixturesTotal int

	if dbClient, err = i.clientFactory(database); err != nil {
		return fixturesTotal, fmt.Errorf("could not create db client: %w", err)
	}

	if res, err = dbClient.GetResult(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?", database); err != nil {
		return fixturesTotal, fmt.Errorf("failed to check tables of database: %w", err)
	}

	for _, row := range *res {
		tables = append(tables, row["TABLE_NAME"])
	}

	for table, rows := range data {
		// discard fixtures for tables that do not exist in destination database
		if !funk.Contains(tables, table) {
			continue
		}

		// truncate table before filling with new data for consistency
		sqls = append(sqls, SqlFmt("TRUNCATE TABLE %s;", []any{table}))
		fixturesTotal += len(rows)

		if stmts, argss, err = i.buildSql(table, rows); err != nil {
			return fixturesTotal, fmt.Errorf("failed to build sqlers for fixture loading: %w", err)
		}

		for i, stmt := range stmts {
			sqls = append(sqls, SqlFmt(stmt, nil, argss[i]...))
		}
	}

	if ress, err = dbClient.ExecMultiInTx(ctx, sqls...); err != nil {
		return fixturesTotal, fmt.Errorf("failed to execute fixture loading queries in transaction: %w", err)
	}

	if len(ress) != len(sqls) {
		return fixturesTotal, fmt.Errorf("expected %d results, got %d", len(sqls), len(ress))
	}

	return fixturesTotal, nil
}

func (i *dataImporter) buildSql(table string, rows []map[string]any) (stmts []string, argss [][]any, err error) {
	if len(rows) == 0 {
		return nil, nil, nil
	}
	squirrelCols := make([]string, 0)
	columns := make([]string, 0)

	for col := range rows[0] {
		squirrelCols = append(squirrelCols, fmt.Sprintf("`%s`", col)) // need to quote these, as they might contain mysql keywords
		columns = append(columns, col)
	}

	insertBuilder := squirrel.
		Insert(table).
		PlaceholderFormat(squirrel.Question).
		Columns(squirrelCols...)
	offset := 0
	var stmt string
	var args []any

	for i, values := range rows {
		// if we exceed the max amount of parameters for a prepared statement, make it a new statement instead
		if ((i+1)*len(squirrelCols))-offset > maxMysqlPreparedStatementArgs {
			offset = i * len(squirrelCols)

			stmt, args, err = insertBuilder.ToSql()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to build sql statement: %w", err)
			}

			stmts = append(stmts, stmt)
			argss = append(argss, args)
			insertBuilder = squirrel.
				Insert(table).
				PlaceholderFormat(squirrel.Question).
				Columns(squirrelCols...)
		}

		valuess := make([]any, len(columns))

		for j, column := range columns {
			valuess[j] = values[column]
		}

		insertBuilder = insertBuilder.Values(valuess...)
	}

	stmt, args, err = insertBuilder.ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build sql statement: %w", err)
	}

	stmts = append(stmts, stmt)
	argss = append(argss, args)

	return stmts, argss, nil
}
