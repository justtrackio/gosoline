package athena

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name Repository
type Repository[T any] interface {
	RepositoryRaw
	QueryBuilder() squirrel.SelectBuilder
	QueryQb(ctx context.Context, qb squirrel.SelectBuilder) ([]*T, error)
	Query(ctx context.Context, query string) ([]*T, error)
}

type repository[T any] struct {
	RepositoryRaw
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
		settings:      settings,
	}
}

func (r *repository[T]) QueryBuilder() squirrel.SelectBuilder {
	columns := r.getColumns()
	qry := squirrel.Select(columns...).From(r.settings.TableName)

	return qry
}

func (r *repository[T]) QueryQb(ctx context.Context, qb squirrel.SelectBuilder) ([]*T, error) {
	var err error
	var sql string
	var args []any

	if sql, args, err = qb.PlaceholderFormat(squirrel.Dollar).ToSql(); err != nil {
		return nil, fmt.Errorf("could not convert query to sql: %w", err)
	}

	if sql, err = ReplaceDollarPlaceholders(sql, args); err != nil {
		return nil, fmt.Errorf("could not replace placeholders: %w", err)
	}

	return r.Query(ctx, sql)
}

func (r *repository[T]) Query(ctx context.Context, query string) ([]*T, error) {
	var err error
	var rows *sqlx.Rows
	var result []*T

	if rows, err = r.QueryRows(ctx, query); err != nil {
		return nil, fmt.Errorf("can not query rows: %w", err)
	}

	for rows.Next() {
		value := new(T)

		if err = rows.StructScan(value); err != nil {
			return nil, fmt.Errorf("could not scan row: %w", err)
		}

		result = append(result, value)
	}

	return result, nil
}

func (r *repository[T]) getColumns() (columns []string) {
	typ := reflect.TypeOf(new(T)).Elem()

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

var replaceRegExp = regexp.MustCompile(`\$\d+`)

// ReplaceDollarPlaceholders replaces $1, $2, etc. with the corresponding element from params. You have to supply
// an additional escape function which converts each parameter to a string safe to embed in the query.
func ReplaceDollarPlaceholders(query string, args []any) (sql string, err error) {
	defer func() {
		if err != nil {
			return
		}

		err = coffin.ResolveRecovery(recover())
	}()

	return replaceRegExp.ReplaceAllStringFunc(query, func(s string) string {
		index, err := strconv.ParseInt(s[1:], 10, 64)
		if err != nil || index < 1 || index > int64(len(args)) {
			return s
		}

		arg := args[index-1]

		switch a := arg.(type) {
		case bool:
			return strconv.FormatBool(a)
		case string, fmt.Stringer:
			return fmt.Sprintf("'%s'", a)
		case []byte:
			return fmt.Sprintf("'%s'", string(a))
		case int, int8, int16, int32, int64:
			return fmt.Sprintf("%d", a)
		case uint, uint8, uint16, uint32, uint64:
			return fmt.Sprintf("%d", a)
		case float32, float64:
			return fmt.Sprintf("%f", a)
		}

		panic(fmt.Sprintf("unexpected type %T", arg))
	}), nil
}
