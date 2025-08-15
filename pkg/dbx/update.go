package dbx

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/lann/builder"
)

type updateData[T any] struct {
	Client db.Client
	Error  error

	PlaceholderFormat PlaceholderFormat
	Prefixes          []Sqlizer
	Table             string
	SetClauses        []setClause
	WhereParts        []Sqlizer
	OrderBys          []string
	Limit             string
	Offset            string
	Suffixes          []Sqlizer
}

type setClause struct {
	column string
	value  any
}

func (d *updateData[T]) Exec(ctx context.Context) (sql.Result, error) {
	if d.Error != nil {
		return nil, fmt.Errorf("unable to execute update query: %w", d.Error)
	}

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

func (d *updateData[T]) toSql() (sqlStr string, args []any, err error) {
	if len(d.Table) == 0 {
		err = fmt.Errorf("update statements must specify a table")
		return
	}
	if len(d.SetClauses) == 0 {
		err = fmt.Errorf("update statements must have at least one Set clause")
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

	sql.WriteString("UPDATE ")
	sql.WriteString(d.Table)

	sql.WriteString(" SET ")
	setSqls := make([]string, len(d.SetClauses))
	for i, setClause := range d.SetClauses {
		var valSql string
		if vs, ok := setClause.value.(Sqlizer); ok {
			vsql, vargs, err := vs.ToSql()
			if err != nil {
				return "", nil, err
			}
			valSql = vsql
			args = append(args, vargs...)
		} else {
			valSql = "?"
			args = append(args, setClause.value)
		}
		setSqls[i] = fmt.Sprintf("%s = %s", setClause.column, valSql)
	}
	sql.WriteString(strings.Join(setSqls, ", "))

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

// UpdateBuilder[T] builds SQL UPDATE statements.
type UpdateBuilder[T any] builder.Builder

func (b UpdateBuilder[T]) Exec(ctx context.Context) (sql.Result, error) {
	data := builder.GetStruct(b).(updateData[T])
	return data.Exec(ctx)
}

// Format methods

// PlaceholderFormat sets PlaceholderFormat (e.g. Question or Dollar) for the
// query.
func (b UpdateBuilder[T]) placeholderFormat(f PlaceholderFormat) UpdateBuilder[T] {
	return builder.Set(b, "PlaceholderFormat", f).(UpdateBuilder[T])
}

// SQL methods

// Prefix adds an expression to the beginning of the query
func (b UpdateBuilder[T]) Prefix(sql string, args ...any) UpdateBuilder[T] {
	return b.PrefixExpr(Expr(sql, args...))
}

// PrefixExpr adds an expression to the very beginning of the query
func (b UpdateBuilder[T]) PrefixExpr(expr Sqlizer) UpdateBuilder[T] {
	return builder.Append(b, "Prefixes", expr).(UpdateBuilder[T])
}

// Table sets the table to be updated.
func (b UpdateBuilder[T]) table(table string) UpdateBuilder[T] {
	return builder.Set(b, "Table", table).(UpdateBuilder[T])
}

// Set adds SET clauses to the query.
func (b UpdateBuilder[T]) Set(column string, value any) UpdateBuilder[T] {
	return builder.Append(b, "SetClauses", setClause{column: column, value: value}).(UpdateBuilder[T])
}

// SetMap is a convenience method which calls .Set for each key/value pair in clauses.
func (b UpdateBuilder[T]) SetMap(clauses map[string]any) UpdateBuilder[T] {
	keys := make([]string, len(clauses))
	i := 0
	for key := range clauses {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	for _, key := range keys {
		val, _ := clauses[key]
		b = b.Set(key, val)
	}
	return b
}

// Where adds WHERE expressions to the query.
//
// See SelectBuilder.Where for more information.
func (b UpdateBuilder[T]) Where(pred any, args ...any) UpdateBuilder[T] {
	return applyWhere[T](b, pred, args...).(UpdateBuilder[T])
}

// OrderBy adds ORDER BY expressions to the query.
func (b UpdateBuilder[T]) OrderBy(orderBys ...string) UpdateBuilder[T] {
	return builder.Extend(b, "OrderBys", orderBys).(UpdateBuilder[T])
}

// Limit sets a LIMIT clause on the query.
func (b UpdateBuilder[T]) Limit(limit uint64) UpdateBuilder[T] {
	return builder.Set(b, "Limit", fmt.Sprintf("%d", limit)).(UpdateBuilder[T])
}

// Offset sets a OFFSET clause on the query.
func (b UpdateBuilder[T]) Offset(offset uint64) UpdateBuilder[T] {
	return builder.Set(b, "Offset", fmt.Sprintf("%d", offset)).(UpdateBuilder[T])
}

// Suffix adds an expression to the end of the query
func (b UpdateBuilder[T]) Suffix(sql string, args ...any) UpdateBuilder[T] {
	return b.SuffixExpr(Expr(sql, args...))
}

// SuffixExpr adds an expression to the end of the query
func (b UpdateBuilder[T]) SuffixExpr(expr Sqlizer) UpdateBuilder[T] {
	return builder.Append(b, "Suffixes", expr).(UpdateBuilder[T])
}
