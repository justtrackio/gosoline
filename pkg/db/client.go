package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	FormatDateTime = "2006-01-02 15:04:05"
)

//go:generate mockery --name SqlResult
type SqlResult interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type (
	ResultRow map[string]string
	Result    []ResultRow
)

//go:generate mockery --name Client
type Client interface {
	GetSingleScalarValue(ctx context.Context, query string, args ...interface{}) (int, error)
	GetResult(ctx context.Context, query string, args ...interface{}) (*Result, error)
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Prepare(ctx context.Context, query string) (*sql.Stmt, error)
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	Queryx(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	WithTx(ctx context.Context, ops *sql.TxOptions, do func(ctx context.Context, tx *sql.Tx) error) error
}

type ClientSqlx struct {
	logger log.Logger
	db     *sqlx.DB
}

func NewClient(config cfg.Config, logger log.Logger, name string) (Client, error) {
	db, err := ProvideConnection(config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not connect to sql database: %w", err)
	}

	return NewClientWithInterfaces(logger, db), nil
}

func NewClientWithSettings(logger log.Logger, settings Settings) (Client, error) {
	db, err := NewConnectionFromSettings(logger, settings)
	if err != nil {
		return nil, fmt.Errorf("can not connect to sql database: %w", err)
	}

	return NewClientWithInterfaces(logger, db), nil
}

func NewClientWithInterfaces(logger log.Logger, db *sqlx.DB) Client {
	return &ClientSqlx{
		logger: logger,
		db:     db,
	}
}

func (c *ClientSqlx) GetSingleScalarValue(ctx context.Context, query string, args ...interface{}) (int, error) {
	var val sql.NullInt64
	err := c.Get(ctx, &val, query, args...)
	if err != nil {
		return 0, err
	}

	if !val.Valid {
		return 0, nil
	}

	return int(val.Int64), err
}

func (c *ClientSqlx) GetResult(ctx context.Context, query string, args ...interface{}) (*Result, error) {
	out := make(Result, 0, 32)
	rows, err := c.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	cols, _ := rows.Columns()
	types := make(map[string]string)

	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))

		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		m := make(ResultRow)
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})

			if _, ok := types[colName]; !ok {
				types[colName] = reflect.TypeOf(*val).String()
			}

			switch types[colName] {
			case "string":
				m[colName] = (*val).(string)
			case "[]uint8":
				m[colName] = string((*val).([]uint8))
			case "int":
				m[colName] = strconv.FormatInt(int64((*val).(int)), 10)
			case "int64":
				m[colName] = strconv.FormatInt((*val).(int64), 10)
			case "float64":
				m[colName] = strconv.FormatFloat((*val).(float64), 'f', -1, 64)
			default:
				errStr := fmt.Sprintf("could not convert mysql result into string map: %v -> %v is %v", colName, *val, reflect.TypeOf(*val))
				return nil, errors.New(errStr)
			}
		}

		out = append(out, m)
	}

	return &out, err
}

func (c *ClientSqlx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	c.logger.Debug("> %s %q", query, args)

	return c.db.ExecContext(ctx, query, args...)
}

func (c *ClientSqlx) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	return c.db.PrepareContext(ctx, query)
}

func (c *ClientSqlx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	c.logger.Debug("> %s %q", query, args)

	return c.db.QueryContext(ctx, query, args...)
}

func (c *ClientSqlx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

func (c *ClientSqlx) Queryx(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	c.logger.Debug("> %s %q", query, args)

	return c.db.QueryxContext(ctx, query, args...)
}

func (c *ClientSqlx) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	c.logger.Debug("> %s %q", query, args)

	return c.db.SelectContext(ctx, dest, query, args...)
}

func (c *ClientSqlx) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	c.logger.Debug("> %s %q", query, args)

	return c.db.GetContext(ctx, dest, query, args...)
}

func (c *ClientSqlx) BeginTx(ctx context.Context, ops *sql.TxOptions) (*sql.Tx, error) {
	c.logger.Debug("start tx")

	return c.db.BeginTx(ctx, ops)
}

func (c *ClientSqlx) WithTx(ctx context.Context, ops *sql.TxOptions, do func(ctx context.Context, tx *sql.Tx) error) (err error) {
	var tx *sql.Tx
	tx, err = c.BeginTx(ctx, ops)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			errRollback := tx.Rollback()
			if errRollback != nil {
				err = multierror.Append(err, fmt.Errorf("can not roolback tx: %w", errRollback))
				return
			}
			c.logger.WithContext(ctx).Debug("rollback successfully done")
		}
	}()

	err = do(ctx, tx)
	if err != nil {
		return fmt.Errorf("can not execute do function: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("can not commit tx: %w", err)
	}

	return nil
}
