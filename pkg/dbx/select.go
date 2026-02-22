package dbx

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/lann/builder"
)

type Sqlizer interface {
	ToSql() (string, []any, error)
}

type rawSqlizer interface {
	toSqlRaw() (string, []any, error)
}

type selectData[T any] struct {
	Client db.Client

	PlaceholderFormat PlaceholderFormat
	Options           []string
	Columns           []Sqlizer
	CustomColumns     bool // tracks if columns were explicitly set via Column()
	From              Sqlizer
	Joins             []Sqlizer
	WhereParts        []Sqlizer
	GroupBys          []string
	HavingParts       []Sqlizer
	OrderByParts      []Sqlizer
	Limit             string
	Offset            string
}

func (d *selectData[T]) Exec(ctx context.Context) ([]T, error) {
	var err error
	var sql string
	var args []any

	if sql, args, err = d.toSql(); err != nil {
		return nil, fmt.Errorf("unable to build select query: %w", err)
	}

	dest := make([]T, 0)
	if err = d.Client.Select(ctx, &dest, sql, args...); err != nil {
		return nil, fmt.Errorf("unable to execute select query: %w", err)
	}

	return dest, nil
}

func (d *selectData[T]) toSql() (sqlStr string, args []any, err error) {
	sqlStr, args, err = d.toSqlRaw()
	if err != nil {
		return
	}

	sqlStr, err = d.PlaceholderFormat.ReplacePlaceholders(sqlStr)

	return
}

//nolint:gocognit // copied from external: candidate for a refactoring
func (d *selectData[T]) toSqlRaw() (sqlStr string, args []any, err error) {
	if len(d.Columns) == 0 {
		err = fmt.Errorf("select statements must have at least one result column")

		return
	}

	sql := &bytes.Buffer{}
	sql.WriteString("SELECT ")

	if len(d.Options) > 0 {
		sql.WriteString(strings.Join(d.Options, " "))
		sql.WriteString(" ")
	}

	if len(d.Columns) > 0 {
		args, err = appendToSql(d.Columns, sql, ", ", args)
		if err != nil {
			return
		}
	}

	if d.From != nil {
		sql.WriteString(" FROM ")
		args, err = appendToSql([]Sqlizer{d.From}, sql, "", args)
		if err != nil {
			return
		}
	}

	if len(d.Joins) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(d.Joins, sql, " ", args)
		if err != nil {
			return
		}
	}

	if len(d.WhereParts) > 0 {
		sql.WriteString(" WHERE ")
		args, err = appendToSql(d.WhereParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(d.GroupBys) > 0 {
		sql.WriteString(" GROUP BY ")
		sql.WriteString(strings.Join(d.GroupBys, ", "))
	}

	if len(d.HavingParts) > 0 {
		sql.WriteString(" HAVING ")
		args, err = appendToSql(d.HavingParts, sql, " AND ", args)
		if err != nil {
			return
		}
	}

	if len(d.OrderByParts) > 0 {
		sql.WriteString(" ORDER BY ")
		args, err = appendToSql(d.OrderByParts, sql, ", ", args)
		if err != nil {
			return
		}
	}

	if d.Limit != "" {
		sql.WriteString(" LIMIT ")
		sql.WriteString(d.Limit)
	}

	if d.Offset != "" {
		sql.WriteString(" OFFSET ")
		sql.WriteString(d.Offset)
	}

	sqlStr = sql.String()

	return
}

// Builder

// SelectBuilder[T] builds SQL SELECT statements.
type SelectBuilder[T any] builder.Builder

func newSelectBuilder[T any](client db.Client, table string, placeholderFormat PlaceholderFormat) SelectBuilder[T] {
	b := builder.EmptyBuilder
	sb := SelectBuilder[T](b).from(table)
	sb = sb.placeholderFormat(placeholderFormat)
	sb = builder.Set(sb, "Client", client).(SelectBuilder[T])

	return sb
}

func (b SelectBuilder[T]) Exec(ctx context.Context) ([]T, error) {
	data := builder.GetStruct(b).(selectData[T])

	return data.Exec(ctx)
}

// Format methods

// PlaceholderFormat sets PlaceholderFormat (e.g. Question or Dollar) for the
// query.
func (b SelectBuilder[T]) placeholderFormat(f PlaceholderFormat) SelectBuilder[T] {
	return builder.Set(b, "PlaceholderFormat", f).(SelectBuilder[T])
}

// SQL methods

// Distinct adds a DISTINCT clause to the query.
func (b SelectBuilder[T]) Distinct() SelectBuilder[T] {
	return b.Options("DISTINCT")
}

// Options adds select option to the query
func (b SelectBuilder[T]) Options(options ...string) SelectBuilder[T] {
	return builder.Extend(b, "Options", options).(SelectBuilder[T])
}

// Columns adds result columns to the query.
func (b SelectBuilder[T]) columns(columns ...string) SelectBuilder[T] {
	parts := make([]any, 0, len(columns))
	for _, str := range columns {
		parts = append(parts, newPart(quoteIfNeeded(str)))
	}

	return builder.Extend(b, "Columns", parts).(SelectBuilder[T])
}

// Column adds a result column to the query.
// Unlike Columns, Column accepts args which will be bound to placeholders in
// the columns string, for example:
//
//	Column("IF(col IN ("+squirrel.Placeholders(3)+"), 1, 0) as col", 1, 2, 3)
//
// The first call to Column() will clear any default columns and replace them
// with the specified custom columns. Subsequent calls will append.
func (b SelectBuilder[T]) Column(column any, args ...any) SelectBuilder[T] {
	data := builder.GetStruct(b).(selectData[T])

	// If this is the first explicit Column call, clear default columns
	if !data.CustomColumns {
		b = builder.Set(b, "Columns", []any{}).(SelectBuilder[T])
		b = builder.Set(b, "CustomColumns", true).(SelectBuilder[T])
	}

	return builder.Append(b, "Columns", newPart(column, args...)).(SelectBuilder[T])
}

// From sets the FROM clause of the query.
func (b SelectBuilder[T]) from(from string) SelectBuilder[T] {
	return builder.Set(b, "From", newPart(quoteIfNeeded(from))).(SelectBuilder[T])
}

// JoinClause adds a join clause to the query.
func (b SelectBuilder[T]) JoinClause(pred any, args ...any) SelectBuilder[T] {
	return builder.Append(b, "Joins", newPart(pred, args...)).(SelectBuilder[T])
}

// Join adds a JOIN clause to the query.
func (b SelectBuilder[T]) Join(join string, rest ...any) SelectBuilder[T] {
	return b.JoinClause("JOIN "+join, rest...)
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (b SelectBuilder[T]) LeftJoin(join string, rest ...any) SelectBuilder[T] {
	return b.JoinClause("LEFT JOIN "+join, rest...)
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (b SelectBuilder[T]) RightJoin(join string, rest ...any) SelectBuilder[T] {
	return b.JoinClause("RIGHT JOIN "+join, rest...)
}

// InnerJoin adds a INNER JOIN clause to the query.
func (b SelectBuilder[T]) InnerJoin(join string, rest ...any) SelectBuilder[T] {
	return b.JoinClause("INNER JOIN "+join, rest...)
}

// CrossJoin adds a CROSS JOIN clause to the query.
func (b SelectBuilder[T]) CrossJoin(join string, rest ...any) SelectBuilder[T] {
	return b.JoinClause("CROSS JOIN "+join, rest...)
}

// Where adds an expression to the WHERE clause of the query.
//
// Expressions are ANDed together in the generated SQL.
//
// Where accepts several types for its pred argument:
//
// nil OR "" - ignored.
//
// string - SQL expression.
// If the expression has SQL placeholders then a set of arguments must be passed
// as well, one for each placeholder.
//
// map[string]any OR Eq - map of SQL expressions to values. Each key is
// transformed into an expression like "<key> = ?", with the corresponding value
// bound to the placeholder. If the value is nil, the expression will be "<key>
// IS NULL". If the value is an array or slice, the expression will be "<key> IN
// (?,?,...)", with one placeholder for each item in the value. These expressions
// are ANDed together.
//
// T - a struct of type T
// The struct will get transformed into a map. Keys with zero values will be ignored.
// The resulting map will be passed to Eq, which results in a handling like with
// map[string]any
//
// Where will panic if pred isn't any of the above types.
func (b SelectBuilder[T]) Where(pred any, args ...any) SelectBuilder[T] {
	return applyWhere[T](b, pred, args...).(SelectBuilder[T])
}

// GroupBy adds GROUP BY expressions to the query.
func (b SelectBuilder[T]) GroupBy(groupBys ...string) SelectBuilder[T] {
	return builder.Extend(b, "GroupBys", groupBys).(SelectBuilder[T])
}

// Having adds an expression to the HAVING clause of the query.
//
// See Where.
func (b SelectBuilder[T]) Having(pred any, rest ...any) SelectBuilder[T] {
	return builder.Append(b, "HavingParts", newWherePart(pred, rest...)).(SelectBuilder[T])
}

// OrderByClause adds ORDER BY clause to the query.
func (b SelectBuilder[T]) OrderByClause(pred any, args ...any) SelectBuilder[T] {
	return builder.Append(b, "OrderByParts", newPart(pred, args...)).(SelectBuilder[T])
}

// OrderBy adds ORDER BY expressions to the query.
func (b SelectBuilder[T]) OrderBy(orderBys ...string) SelectBuilder[T] {
	for _, orderBy := range orderBys {
		b = b.OrderByClause(orderBy)
	}

	return b
}

// Limit sets a LIMIT clause on the query.
func (b SelectBuilder[T]) Limit(limit uint64) SelectBuilder[T] {
	return builder.Set(b, "Limit", fmt.Sprintf("%d", limit)).(SelectBuilder[T])
}

// Limit ALL allows to access all records with limit
func (b SelectBuilder[T]) RemoveLimit() SelectBuilder[T] {
	return builder.Delete(b, "Limit").(SelectBuilder[T])
}

// Offset sets a OFFSET clause on the query.
func (b SelectBuilder[T]) Offset(offset uint64) SelectBuilder[T] {
	return builder.Set(b, "Offset", fmt.Sprintf("%d", offset)).(SelectBuilder[T])
}

// RemoveOffset removes OFFSET clause.
func (b SelectBuilder[T]) RemoveOffset() SelectBuilder[T] {
	return builder.Delete(b, "Offset").(SelectBuilder[T])
}
