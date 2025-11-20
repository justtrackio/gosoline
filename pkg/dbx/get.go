package dbx

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/lann/builder"
)

// getData[T] reuses the selectData fields and SQL generation but changes Exec semantics.
type getData[T any] struct {
	selectData[T]
}

func (d *getData[T]) Exec(ctx context.Context) (T, error) {
	var zero T

	// Enforce a small LIMIT to detect "too many" results.
	d.Limit = "2"

	sqlStr, args, err := d.toSql()
	if err != nil {
		return zero, fmt.Errorf("unable to build get query: %w", err)
	}

	dest := make([]T, 0, 2)
	if err = d.Client.Select(ctx, &dest, sqlStr, args...); err != nil {
		return zero, fmt.Errorf("unable to execute get query: %w", err)
	}

	switch len(dest) {
	case 0:
		return zero, ErrNotFound
	case 1:
		return dest[0], nil
	default:
		return zero, fmt.Errorf("dbx: expected 1 result, got %d", len(dest))
	}
}

// GetBuilder[T] builds SELECT ... LIMIT 2 statements which return one element.
type GetBuilder[T any] builder.Builder

func newGetBuilder[T any](client db.Client, table string, placeholderFormat PlaceholderFormat) GetBuilder[T] {
	b := builder.Builder(builder.EmptyBuilder)
	gb := GetBuilder[T](b).from(table)
	gb = gb.placeholderFormat(placeholderFormat)
	gb = builder.Set(gb, "Client", client).(GetBuilder[T])

	return gb
}

func (b GetBuilder[T]) Exec(ctx context.Context) (T, error) {
	data := builder.GetStruct(b).(getData[T])

	return data.Exec(ctx)
}

// Format methods

// placeholderFormat sets PlaceholderFormat (e.g. Question or Dollar) for the query.
func (b GetBuilder[T]) placeholderFormat(f PlaceholderFormat) GetBuilder[T] {
	return builder.Set(b, "PlaceholderFormat", f).(GetBuilder[T])
}

// SQL methods

// Options adds select options to the query (e.g. SQL_NO_CACHE, HIGH_PRIORITY).
func (b GetBuilder[T]) Options(options ...string) GetBuilder[T] {
	return builder.Extend(b, "Options", options).(GetBuilder[T])
}

// columns adds result columns to the query (used internally by client.Get()).
func (b GetBuilder[T]) columns(columns ...string) GetBuilder[T] {
	parts := make([]any, 0, len(columns))
	for _, str := range columns {
		parts = append(parts, newPart(str))
	}

	return builder.Extend(b, "Columns", parts).(GetBuilder[T])
}

// Column adds a result column to the query.
// See SelectBuilder.Column for details.
func (b GetBuilder[T]) Column(column any, args ...any) GetBuilder[T] {
	return builder.Append(b, "Columns", newPart(column, args...)).(GetBuilder[T])
}

// from sets the FROM clause of the query.
func (b GetBuilder[T]) from(from string) GetBuilder[T] {
	return builder.Set(b, "From", newPart(from)).(GetBuilder[T])
}

// JoinClause adds a join clause to the query.
func (b GetBuilder[T]) JoinClause(pred any, args ...any) GetBuilder[T] {
	return builder.Append(b, "Joins", newPart(pred, args...)).(GetBuilder[T])
}

// Join adds a JOIN clause to the query.
func (b GetBuilder[T]) Join(join string, rest ...any) GetBuilder[T] {
	return b.JoinClause("JOIN "+join, rest...)
}

// LeftJoin adds a LEFT JOIN clause to the query.
func (b GetBuilder[T]) LeftJoin(join string, rest ...any) GetBuilder[T] {
	return b.JoinClause("LEFT JOIN "+join, rest...)
}

// RightJoin adds a RIGHT JOIN clause to the query.
func (b GetBuilder[T]) RightJoin(join string, rest ...any) GetBuilder[T] {
	return b.JoinClause("RIGHT JOIN "+join, rest...)
}

// InnerJoin adds an INNER JOIN clause to the query.
func (b GetBuilder[T]) InnerJoin(join string, rest ...any) GetBuilder[T] {
	return b.JoinClause("INNER JOIN "+join, rest...)
}

// CrossJoin adds a CROSS JOIN clause to the query.
func (b GetBuilder[T]) CrossJoin(join string, rest ...any) GetBuilder[T] {
	return b.JoinClause("CROSS JOIN "+join, rest...)
}

// Where adds an expression to the WHERE clause of the query.
//
// See SelectBuilder.Where for the supported predicate types (string, Eq/map,
// struct T, etc.).
func (b GetBuilder[T]) Where(pred any, args ...any) GetBuilder[T] {
	return applyWhere[T](b, pred, args...).(GetBuilder[T])
}

// GroupBy adds GROUP BY expressions to the query.
func (b GetBuilder[T]) GroupBy(groupBys ...string) GetBuilder[T] {
	return builder.Extend(b, "GroupBys", groupBys).(GetBuilder[T])
}

// Having adds an expression to the HAVING clause of the query.
//
// See SelectBuilder.Where for details.
func (b GetBuilder[T]) Having(pred any, rest ...any) GetBuilder[T] {
	return builder.Append(b, "HavingParts", newWherePart(pred, rest...)).(GetBuilder[T])
}

// OrderByClause adds ORDER BY clause to the query.
func (b GetBuilder[T]) OrderByClause(pred any, args ...any) GetBuilder[T] {
	return builder.Append(b, "OrderByParts", newPart(pred, args...)).(GetBuilder[T])
}

// OrderBy adds ORDER BY expressions to the query.
func (b GetBuilder[T]) OrderBy(orderBys ...string) GetBuilder[T] {
	for _, orderBy := range orderBys {
		b = b.OrderByClause(orderBy)
	}

	return b
}
