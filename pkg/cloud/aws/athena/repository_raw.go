package athena

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/go-athena"
)

type Settings struct {
	ClientName string
	TableName  string
}

// RepositoryRaw is a low level repository implementation to query Athena tables.
//
//go:generate mockery --name RepositoryRaw
type RepositoryRaw interface {
	// QueryRows accepts a SQL statement and returns the result as *sqlx.Rows
	QueryRows(ctx context.Context, sql string) (*sqlx.Rows, error)
}

type repositoryRaw struct {
	db       *sqlx.DB
	settings *Settings
}

func NewRepositoryRaw(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (*repositoryRaw, error) {
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

	return NewRepositoryRawWithInterfaces(db, settings), nil
}

func NewRepositoryRawWithInterfaces(db *sql.DB, settings *Settings) *repositoryRaw {
	return &repositoryRaw{
		db:       sqlx.NewDb(db, "athena"),
		settings: settings,
	}
}

func (r *repositoryRaw) QueryRows(ctx context.Context, sql string) (*sqlx.Rows, error) {
	var err error
	var rows *sqlx.Rows

	if rows, err = r.db.QueryxContext(ctx, sql); err != nil {
		return nil, fmt.Errorf("executing query %s threw an error: %w", sql, err)
	}

	return rows, nil
}
