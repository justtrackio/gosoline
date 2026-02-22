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
	Options           []string
	Table             string
	SetClauses        []setClause
	WhereParts        []Sqlizer
	OrderBys          []string
	Limit             string
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
		return nil, fmt.Errorf("unable to execute update query: %w", err)
	}

	return res, nil
}

func (d *updateData[T]) toSql() (sqlStr string, args []any, err error) {
	if d.Table == "" {
		err = fmt.Errorf("update statements must specify a table")

		return
	}
	if len(d.SetClauses) == 0 {
		err = fmt.Errorf("update statements must have at least one Set clause")

		return
	}

	sql := &bytes.Buffer{}
	sql.WriteString("UPDATE ")

	if len(d.Options) > 0 {
		sql.WriteString(strings.Join(d.Options, " "))
		sql.WriteString(" ")
	}

	sql.WriteString(quoteIfNeeded(d.Table))

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
		setSqls[i] = fmt.Sprintf("%s = %s", quoteIfNeeded(setClause.column), valSql)
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

	if d.Limit != "" {
		sql.WriteString(" LIMIT ")
		sql.WriteString(d.Limit)
	}

	sqlStr, err = d.PlaceholderFormat.ReplacePlaceholders(sql.String())

	return
}

// Builder

// UpdateBuilder[T] builds SQL UPDATE statements.
type UpdateBuilder[T any] builder.Builder

func newUpdateBuilder[T any](client db.Client, table string, placeholderFormat PlaceholderFormat) UpdateBuilder[T] {
	b := builder.Builder(builder.EmptyBuilder)
	ub := UpdateBuilder[T](b).table(table)
	ub = ub.placeholderFormat(placeholderFormat)
	ub = builder.Set(ub, "Client", client).(UpdateBuilder[T])

	return ub
}

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

// Table sets the table to be updated.
func (b UpdateBuilder[T]) table(table string) UpdateBuilder[T] {
	return builder.Set(b, "Table", table).(UpdateBuilder[T])
}

// Set adds SET clauses to the query.
func (b UpdateBuilder[T]) Set(column string, value any) UpdateBuilder[T] {
	return builder.Append(b, "SetClauses", setClause{column: column, value: value}).(UpdateBuilder[T])
}

// Set adds SET clauses to the query.
func (b UpdateBuilder[T]) Options(options ...string) UpdateBuilder[T] {
	return builder.Extend(b, "Options", options).(UpdateBuilder[T])
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
		val := clauses[key]
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
