package athena

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/go-athena"
)

var dateInvervals = []time.Duration{
	time.Hour * 24,
	time.Hour * 24 * 7,
	time.Hour * 24 * 30,
}

type AthenaSettings struct {
	ClientName string
	TableName  string
}

//go:generate mockery --name AthenaRepository
type AthenaRepository[T any] interface {
	QueryBuilder() squirrel.SelectBuilder
	RunQuery(query string) ([]*T, error)
	SearchInIntervals(baseQuery squirrel.SelectBuilder, untilDate time.Time, limit uint) ([]*T, error)
}

type athenaRepository[T any] struct {
	db       *sqlx.DB
	settings *AthenaSettings
}

func NewAthenaRepository[T any](ctx context.Context, config cfg.Config, logger log.Logger, settings *AthenaSettings) (*athenaRepository[T], error) {
	var err error
	var clientCfg *ClientConfig
	var awsConfig aws.Config

	if clientCfg, awsConfig, err = getConfigs(ctx, config, logger, settings.ClientName); err != nil {
		return nil, fmt.Errorf("can not initialize config: %w", err)
	}

	db, err := athena.Open(athena.DriverConfig{
		Config:         &awsConfig,
		Database:       clientCfg.Settings.Database,
		OutputLocation: clientCfg.Settings.OutputLocation,
		PollFrequency:  clientCfg.Settings.PollFrequency,
	})
	if err != nil {
		return nil, fmt.Errorf("could not open Athena connection: %w", err)
	}

	return NewAthenaRepositoryWithInterfaces[T](db, settings), nil
}

func NewAthenaRepositoryWithInterfaces[T any](db *sql.DB, settings *AthenaSettings) *athenaRepository[T] {
	return &athenaRepository[T]{
		db:       sqlx.NewDb(db, "athena"),
		settings: settings,
	}
}

func (r *athenaRepository[T]) QueryBuilder() squirrel.SelectBuilder {
	columns := r.getColumns()
	qry := squirrel.Select(columns...).From(r.settings.TableName)

	return qry
}

func (r *athenaRepository[T]) RunQuery(query string) ([]*T, error) {
	var err error
	var rows *sqlx.Rows
	var result []*T

	if rows, err = r.db.Queryx(query); err != nil {
		return nil, fmt.Errorf("executing query %s threw an error: %w", query, err)
	}

	for rows.Next() {
		value := new(T)

		if err = rows.StructScan(value); err != nil {
			return nil, fmt.Errorf("could not scan row: %w", err)
		}

		result = append(result, value)
	}

	return result, nil
}

func (r *athenaRepository[T]) getColumns() (columns []string) {
	typ := reflect.TypeOf(new(T)).Elem()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		column, ok := field.Tag.Lookup("db")

		if !ok {
			continue
		}

		columns = append(columns, column)
	}

	return
}
