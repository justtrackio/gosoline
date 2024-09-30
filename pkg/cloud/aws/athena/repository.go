package athena

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// Repository is a high level repository implementation to query Athena.
// The repository is typed to return a slice of structs as a result instead of raw rows.
//
//go:generate mockery --name Repository
type Repository[T any] interface {
	RepositoryRaw
	// QueryBuilder returns a prepared query builder with prefilled columns and table name
	QueryBuilder() squirrel.SelectBuilder
	// Query accepts a query builder and executes the resulting sql statement.
	Query(ctx context.Context, qb squirrel.SelectBuilder) ([]T, error)
	// QuerySql accepts an already formatted sql statement and executes it.
	QuerySql(ctx context.Context, query string) ([]T, error)
}

type repository[T any] struct {
	RepositoryRaw
	columns  []string
	settings *Settings
}

func NewRepository[T any](ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (*repository[T], error) {
	var err error
	var raw RepositoryRaw

	if raw, err = NewRepositoryRaw(ctx, config, logger, settings); err != nil {
		return nil, fmt.Errorf("can not create raw athena repository %s: %w", settings.TableName, err)
	}

	return NewRepositoryWithInterfaces[T](raw, settings), nil
}

func NewRepositoryWithInterfaces[T any](raw RepositoryRaw, settings *Settings) *repository[T] {
	return &repository[T]{
		RepositoryRaw: raw,
		columns:       getColumns(new(T)),
		settings:      settings,
	}
}

func (r *repository[T]) QueryBuilder() squirrel.SelectBuilder {
	return squirrel.Select(r.columns...).From(r.settings.TableName)
}

func (r *repository[T]) Query(ctx context.Context, qb squirrel.SelectBuilder) ([]T, error) {
	var err error
	var sql string
	var args []any

	if sql, args, err = qb.PlaceholderFormat(squirrel.Dollar).ToSql(); err != nil {
		return nil, fmt.Errorf("could not convert query to sql: %w", err)
	}

	if sql, err = ReplaceDollarPlaceholders(sql, args); err != nil {
		return nil, fmt.Errorf("could not replace placeholders: %w", err)
	}

	return r.QuerySql(ctx, sql)
}

func (r *repository[T]) QuerySql(ctx context.Context, query string) (result []T, err error) {
	var rows *sqlx.Rows

	if rows, err = r.QueryRows(ctx, query); err != nil {
		err = fmt.Errorf("can not query rows: %w", err)

		return
	}

	defer func() {
		if err = rows.Close(); err != nil {
			err = fmt.Errorf("can not close rows: %w", err)
		}
	}()

	for rows.Next() {
		value := new(T)

		if err = rows.StructScan(value); err != nil {
			err = fmt.Errorf("could not scan row: %w", err)

			return
		}

		result = append(result, *value)
	}

	if rows.Err() != nil {
		err = fmt.Errorf("could not scan rows: %w", rows.Err())

		return
	}

	return
}

func getColumns(val any) (columns []string) {
	typ := reflect.TypeOf(val).Elem()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		column, ok := field.Tag.Lookup("db")

		if !ok {
			continue
		}

		columns = append(columns, column)
	}

	return
}
