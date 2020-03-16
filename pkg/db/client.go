package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/VividCortex/mysqlerr"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/cenkalti/backoff"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"reflect"
	"strconv"
)

const (
	FormatDateTime                 = "2006-01-02 15:04:05"
	mysqlDuplicateEntryErrorNumber = 1062
)

type ResultRow map[string]string
type Result []ResultRow

//go:generate mockery -name Client
type Client interface {
	GetSingleScalarValue(query string, args ...interface{}) (int, error)
	GetResult(query string, args ...interface{}) (*Result, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Select(dest interface{}, query string, args ...interface{}) error
	Get(dest interface{}, query string, args ...interface{}) error
}

type ClientSqlx struct {
	logger mon.Logger
	db     *sqlx.DB
}

func NewClient(config cfg.Config, logger mon.Logger) Client {
	db, err := ProvideDefaultConnection(config, logger)

	if err != nil {
		logger.Fatal(err, "can not connect to sql database")
	}

	return NewClientWithInterfaces(logger, db)
}

//func NewClientWithDefault(logger mon.Logger) Client {
//	logger = logger.WithChannel("sql")
//	db, err := ProvideDefaultConnection()
//
//	return NewClient(logger, DefaultConnection)
//}

func NewClientWithInterfaces(logger mon.Logger, db *sqlx.DB) Client {
	if db == nil {
		logger.WithContext(context.Background()).Fatal(errors.New("db not booted yet"), "db not booted yet")
	}

	return &ClientSqlx{
		logger: logger.WithContext(context.Background()), // TODO: this is not nice, but we don't (yet) have a context when logging in this module
		db:     db,
	}
}

func (c *ClientSqlx) GetSingleScalarValue(query string, args ...interface{}) (int, error) {
	var val sql.NullInt64
	err := c.Get(&val, query, args...)

	if err != nil {
		return 0, err
	}

	if !val.Valid {
		return 0, nil
	}

	return int(val.Int64), err
}

func (c *ClientSqlx) GetResult(query string, args ...interface{}) (*Result, error) {
	out := make(Result, 0, 32)
	rows, err := c.Query(query, args...)

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
				m[colName] = string((*val).(string))
			case "[]uint8":
				m[colName] = string((*val).([]uint8))
			case "int":
				m[colName] = string((*val).(int))
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

func (c *ClientSqlx) Exec(query string, args ...interface{}) (sql.Result, error) {
	c.logger.Debugf("> %s %q", query, args)

	return c.db.Exec(query, args...)
}

func (c *ClientSqlx) Prepare(query string) (*sql.Stmt, error) {
	return c.db.Prepare(query)
}

func (c *ClientSqlx) Query(query string, args ...interface{}) (*sql.Rows, error) {
	c.logger.Debugf("> %s %q", query, args)

	return c.db.Query(query, args...)
}

func (c *ClientSqlx) QueryRow(query string, args ...interface{}) *sql.Row {
	return c.db.QueryRow(query, args)
}

func (c *ClientSqlx) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	c.logger.Debugf("> %s %q", query, args)

	return c.db.Queryx(query, args...)
}

func (c *ClientSqlx) Select(dest interface{}, query string, args ...interface{}) error {
	c.logger.Debugf("> %s %q", query, args)

	return c.db.Select(dest, query, args...)
}

func (c *ClientSqlx) Get(dest interface{}, query string, args ...interface{}) error {
	c.logger.Debugf("> %s %q", query, args)

	return c.db.Get(dest, query, args...)
}

func (c *ClientSqlx) execWithBackoff(query string, args ...interface{}) (res sql.Result, err error) {
	backoffConfig := backoff.NewExponentialBackOff()

	err = backoff.Retry(func() error {
		res, err = c.db.Exec(query, args...)

		mysqlE, ok := err.(*mysql.MySQLError)

		if ok && mysqlE.Number != mysqlerr.ER_LOCK_DEADLOCK {
			return nil
		}

		return err
	}, backoffConfig)

	if err != nil {
		c.logger.Error(err, "could not execute query with backoff")
	}

	return
}

func IsDuplicateEntryError(err error) bool {
	if err != nil {
		mysqlErr, ok := err.(*mysql.MySQLError)

		return ok && mysqlErr.Number == mysqlDuplicateEntryErrorNumber
	}

	return false
}
