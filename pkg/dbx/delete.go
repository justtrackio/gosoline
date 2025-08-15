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
	Prefixes          []Sqlizer
	From              string
	WhereParts        []Sqlizer
	OrderBys          []string
	Limit             string
	Offset            string
	Suffixes          []Sqlizer
}

func (d *deleteData[T]) Exec(ctx context.Context) (sql.Result, error) {
	var err error
	var res sql.Result
	var sql string
	var args []any

	if sql, args, err = d.toSql(); err != nil {
		return nil, fmt.Errorf("unable to build update query: %w", err)
	}

	if res, err = d.Client.Exec(ctx, sql, args...); err != nil {
		return nil, fmt.Errorf("unable to execute select query: %w", err)
	}

	return res, nil
}

func (d *deleteData[T]) toSql() (sqlStr string, args []any, err error) {
	if len(d.From) == 0 {
		err = fmt.Errorf("delete statements must specify a From table")
		return
	}

	sql := &bytes.Buffer{}

	if len(d.Prefixes) > 0 {
		args, err = appendToSql(d.Prefixes, sql, " ", args)
		if err != nil {
			return
		}

		sql.WriteString(" ")
	}

	sql.WriteString("DELETE FROM ")
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

	if len(d.Limit) > 0 {
		sql.WriteString(" LIMIT ")
		sql.WriteString(d.Limit)
	}

	if len(d.Offset) > 0 {
		sql.WriteString(" OFFSET ")
		sql.WriteString(d.Offset)
	}

	if len(d.Suffixes) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(d.Suffixes, sql, " ", args)
		if err != nil {
			return
		}
	}

	sqlStr, err = d.PlaceholderFormat.ReplacePlaceholders(sql.String())
	return
}

// Builder

// DeleteBuilder[T] builds SQL DELETE statements.
type DeleteBuilder[T any] builder.Builder

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

// Prefix adds an expression to the beginning of the query
func (b DeleteBuilder[T]) Prefix(sql string, args ...any) DeleteBuilder[T] {
	return b.PrefixExpr(Expr(sql, args...))
}

// PrefixExpr adds an expression to the very beginning of the query
func (b DeleteBuilder[T]) PrefixExpr(expr Sqlizer) DeleteBuilder[T] {
	return builder.Append(b, "Prefixes", expr).(DeleteBuilder[T])
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

// Offset sets a OFFSET clause on the query.
func (b DeleteBuilder[T]) Offset(offset uint64) DeleteBuilder[T] {
	return builder.Set(b, "Offset", fmt.Sprintf("%d", offset)).(DeleteBuilder[T])
}

// Suffix adds an expression to the end of the query
func (b DeleteBuilder[T]) Suffix(sql string, args ...any) DeleteBuilder[T] {
	return b.SuffixExpr(Expr(sql, args...))
}

// SuffixExpr adds an expression to the end of the query
func (b DeleteBuilder[T]) SuffixExpr(expr Sqlizer) DeleteBuilder[T] {
	return builder.Append(b, "Suffixes", expr).(DeleteBuilder[T])
}
