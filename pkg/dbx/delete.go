package dbx

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/lann/builder"
)

type deleteData[T any] struct {
	Client db.Client

	PlaceholderFormat PlaceholderFormat
	Options           []string
	From              string
	WhereParts        []Sqlizer
	OrderBys          []string
	Limit             string
}

func (d *deleteData[T]) Exec(ctx context.Context) (sql.Result, error) {
	var err error
	var res sql.Result
	var sql string
	var args []any

	if sql, args, err = d.toSql(); err != nil {
		return nil, fmt.Errorf("unable to build delete query: %w", err)
	}

	if res, err = d.Client.Exec(ctx, sql, args...); err != nil {
		return nil, fmt.Errorf("unable to execute delete query: %w", err)
	}

	return res, nil
}

func (d *deleteData[T]) toSql() (sqlStr string, args []any, err error) {
	if d.From == "" {
		err = fmt.Errorf("delete statements must specify a From table")

		return
	}

	sql := &bytes.Buffer{}
	sql.WriteString("DELETE ")
	if len(d.Options) > 0 {
		sql.WriteString(strings.Join(d.Options, " "))
		sql.WriteString(" ")
	}

	sql.WriteString("FROM ")
	sql.WriteString(d.From)

	if len(d.WhereParts) > 0 {
		sql.WriteString(" WHERE ")
		args, err = appendToSql(d.WhereParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(d.OrderBys) > 0 {
		sql.WriteString(" ORDER BY ")
		sql.WriteString(strings.Join(d.OrderBys, ", "))
	}

	if d.Limit != "" {
		sql.WriteString(" LIMIT ")
		sql.WriteString(d.Limit)
	}

	sqlStr, err = d.PlaceholderFormat.ReplacePlaceholders(sql.String())

	return
}

// Builder

// DeleteBuilder[T] builds SQL DELETE statements.
type DeleteBuilder[T any] builder.Builder

func newDeleteBuilder[T any](client db.Client, table string, placeholderFormat PlaceholderFormat) DeleteBuilder[T] {
	b := builder.Builder(builder.EmptyBuilder)
	db := DeleteBuilder[T](b).from(table)
	db = db.placeholderFormat(placeholderFormat)
	db = builder.Set(db, "Client", client).(DeleteBuilder[T])

	return db
}

// Format methods

// PlaceholderFormat sets PlaceholderFormat (e.g. Question or Dollar) for the
// query.
func (b DeleteBuilder[T]) placeholderFormat(f PlaceholderFormat) DeleteBuilder[T] {
	return builder.Set(b, "PlaceholderFormat", f).(DeleteBuilder[T])
}

// Runner methods

// Exec builds and Execs the query with the Runner set by RunWith.
func (b DeleteBuilder[T]) Exec(ctx context.Context) (sql.Result, error) {
	data := builder.GetStruct(b).(deleteData[T])

	return data.Exec(ctx)
}

// SQL methods

// Options adds keyword options before the INTO clause of the query.
func (b DeleteBuilder[T]) Options(options ...string) DeleteBuilder[T] {
	return builder.Extend(b, "Options", options).(DeleteBuilder[T])
}

// From sets the table to be deleted from.
func (b DeleteBuilder[T]) from(from string) DeleteBuilder[T] {
	return builder.Set(b, "From", from).(DeleteBuilder[T])
}

// Where adds WHERE expressions to the query.
//
// See SelectBuilder.Where for more information.
func (b DeleteBuilder[T]) Where(pred any, args ...any) DeleteBuilder[T] {
	return applyWhere[T](b, pred, args...).(DeleteBuilder[T])
}

// OrderBy adds ORDER BY expressions to the query.
func (b DeleteBuilder[T]) OrderBy(orderBys ...string) DeleteBuilder[T] {
	return builder.Extend(b, "OrderBys", orderBys).(DeleteBuilder[T])
}

// Limit sets a LIMIT clause on the query.
func (b DeleteBuilder[T]) Limit(limit uint64) DeleteBuilder[T] {
	return builder.Set(b, "Limit", fmt.Sprintf("%d", limit)).(DeleteBuilder[T])
}
