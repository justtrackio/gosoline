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

	StatementKeyword string
	Options          []string
	Into             string
	Columns          []string
	Values           []any
	Suffixes         []Sqlizer
}

func (d *insertData[T]) Exec(ctx context.Context) (sql.Result, error) {
	var err error
	var res sql.Result
	var sql string

	if sql, err = d.toSql(); err != nil {
		return nil, fmt.Errorf("unable to build insert query: %w", err)
	}

	if res, err = d.Client.NamedExec(ctx, sql, d.Values); err != nil {
		return nil, fmt.Errorf("unable to execute insert query: %w", err)
	}

	return res, nil
}

func (d *insertData[T]) toSql() (sqlStr string, err error) {
	if d.Into == "" {
		err = errors.New("insert statements must specify a table")

		return
	}

	sql := &bytes.Buffer{}
	args := []any{}

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
		sql.WriteString("(`")
		sql.WriteString(strings.Join(d.Columns, "`,`"))
		sql.WriteString("`) ")
	}

	if err = d.appendValuesToSQL(sql); err != nil {
		return
	}

	if len(d.Suffixes) > 0 {
		sql.WriteString(" ")
		_, err = appendToSql(d.Suffixes, sql, " ", args)
		if err != nil {
			return
		}
	}

	return sql.String(), nil
}

func (d *insertData[T]) appendValuesToSQL(w io.Writer) (err error) {
	if _, err = io.WriteString(w, "VALUES "); err != nil {
		return
	}

	valueStrings := funk.Map(d.Columns, func(c string) string {
		return ":" + c
	})

	if _, err = fmt.Fprintf(w, "(%s)", strings.Join(valueStrings, ",")); err != nil {
		return
	}

	return nil
}

// Builder

// InsertBuilder builds SQL INSERT statements.
type InsertBuilder[T any] builder.Builder

func newInsertBuilder[T any](client db.Client, table string) InsertBuilder[T] {
	b := builder.Builder(builder.EmptyBuilder)
	ib := InsertBuilder[T](b).into(table)
	ib = builder.Set(ib, "Client", client).(InsertBuilder[T])

	return ib
}

func (b InsertBuilder[T]) Exec(ctx context.Context) (sql.Result, error) {
	data := builder.GetStruct(b).(insertData[T])

	return data.Exec(ctx)
}

// SQL methods

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
func (b InsertBuilder[T]) values(value ...T) InsertBuilder[T] {
	return builder.Extend(b, "Values", value).(InsertBuilder[T])
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
