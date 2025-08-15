package dbx

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/lann/builder"
)

type insertData[T any] struct {
	Client db.Client

	Prefixes         []Sqlizer
	StatementKeyword string
	Options          []string
	Into             string
	Columns          []string
	Value            any
	Suffixes         []Sqlizer
}

func (d *insertData[T]) Exec(ctx context.Context) (sql.Result, error) {
	var err error
	var res sql.Result
	var sql string

	if sql, err = d.toSql(); err != nil {
		return nil, fmt.Errorf("unable to build select query: %w", err)
	}

	if res, err = d.Client.NamedExec(ctx, sql, d.Value); err != nil {
		return nil, fmt.Errorf("unable to execute select query: %w", err)
	}

	return res, nil
}

func (d *insertData[T]) toSql() (sqlStr string, err error) {
	if len(d.Into) == 0 {
		err = errors.New("insert statements must specify a table")
		return
	}

	sql := &bytes.Buffer{}
	args := []any{}

	if len(d.Prefixes) > 0 {
		args, err = appendToSql(d.Prefixes, sql, " ", args)
		if err != nil {
			return
		}

		sql.WriteString(" ")
	}

	if d.StatementKeyword == "" {
		sql.WriteString("INSERT ")
	} else {
		sql.WriteString(d.StatementKeyword)
		sql.WriteString(" ")
	}

	if len(d.Options) > 0 {
		sql.WriteString(strings.Join(d.Options, " "))
		sql.WriteString(" ")
	}

	sql.WriteString("INTO ")
	sql.WriteString(d.Into)
	sql.WriteString(" ")

	if len(d.Columns) > 0 {
		sql.WriteString("(")
		sql.WriteString(strings.Join(d.Columns, ","))
		sql.WriteString(") ")
	}

	args, err = d.appendValuesToSQL(sql, args)
	if err != nil {
		return
	}

	if len(d.Suffixes) > 0 {
		sql.WriteString(" ")
		args, err = appendToSql(d.Suffixes, sql, " ", args)
		if err != nil {
			return
		}
	}

	return sql.String(), nil
}

func (d *insertData[T]) appendValuesToSQL(w io.Writer, args []any) ([]any, error) {
	io.WriteString(w, "VALUES ")

	valueStrings := funk.Map(d.Columns, func(c string) string {
		return ":" + c
	})

	io.WriteString(w, fmt.Sprintf("(%s)", strings.Join(valueStrings, ",")))

	return args, nil
}

// Builder

// InsertBuilder builds SQL INSERT statements.
type InsertBuilder[T any] builder.Builder

func (b InsertBuilder[T]) Exec(ctx context.Context) (sql.Result, error) {
	data := builder.GetStruct(b).(insertData[T])
	return data.Exec(ctx)
}

// SQL methods

// Prefix adds an expression to the beginning of the query
func (b InsertBuilder[T]) Prefix(sql string, args ...any) InsertBuilder[T] {
	return b.PrefixExpr(Expr(sql, args...))
}

// PrefixExpr adds an expression to the very beginning of the query
func (b InsertBuilder[T]) PrefixExpr(expr Sqlizer) InsertBuilder[T] {
	return builder.Append(b, "Prefixes", expr).(InsertBuilder[T])
}

// Options adds keyword options before the INTO clause of the query.
func (b InsertBuilder[T]) Options(options ...string) InsertBuilder[T] {
	return builder.Extend(b, "Options", options).(InsertBuilder[T])
}

// Into sets the INTO clause of the query.
func (b InsertBuilder[T]) into(from string) InsertBuilder[T] {
	return builder.Set(b, "Into", from).(InsertBuilder[T])
}

// Columns adds insert columns to the query.
func (b InsertBuilder[T]) columns(columns ...string) InsertBuilder[T] {
	return builder.Extend(b, "Columns", columns).(InsertBuilder[T])
}

// Values adds a single row's values to the query.
func (b InsertBuilder[T]) value(value T) InsertBuilder[T] {
	return builder.Set(b, "Value", value).(InsertBuilder[T])
}

// Suffix adds an expression to the end of the query
func (b InsertBuilder[T]) Suffix(sql string, args ...any) InsertBuilder[T] {
	return b.SuffixExpr(Expr(sql, args...))
}

// SuffixExpr adds an expression to the end of the query
func (b InsertBuilder[T]) SuffixExpr(expr Sqlizer) InsertBuilder[T] {
	return builder.Append(b, "Suffixes", expr).(InsertBuilder[T])
}

func (b InsertBuilder[T]) statementKeyword(keyword string) InsertBuilder[T] {
	return builder.Set(b, "StatementKeyword", keyword).(InsertBuilder[T])
}
