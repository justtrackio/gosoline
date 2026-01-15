package athena

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/segmentio/go-athena"
)

type Settings struct {
	ClientName string
	TableName  string
}

// RepositoryRaw is a low level repository implementation to query Athena tables.
//
//go:generate go run github.com/vektra/mockery/v2 --name RepositoryRaw
type RepositoryRaw interface {
	// QueryRows accepts a SQL statement and returns the result as *sqlx.Rows
	QueryRows(ctx context.Context, sql string) (*sqlx.Rows, error)
}

type repositoryRaw struct {
	db       *sqlx.DB
	executor exec.Executor
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

	res := &exec.ExecutableResource{Type: "cloud/aws/athena", Name: settings.TableName}
	executor := exec.NewBackoffExecutor(logger, res, &clientCfg.Settings.Backoff, []exec.ErrorChecker{
		CheckInternalAthenaError,
	})

	return NewRepositoryRawWithInterfaces(db, executor, settings), nil
}

func NewRepositoryRawWithInterfaces(db *sql.DB, executor exec.Executor, settings *Settings) *repositoryRaw {
	return &repositoryRaw{
		db:       sqlx.NewDb(db, "athena"),
		executor: executor,
		settings: settings,
	}
}

func (r *repositoryRaw) QueryRows(ctx context.Context, sql string) (*sqlx.Rows, error) {
	rows, err := r.executor.Execute(ctx, func(ctx context.Context) (any, error) {
		return r.db.QueryxContext(ctx, sql)
	})
	if err != nil {
		return nil, fmt.Errorf("executing query %s threw an error: %w", sql, err)
	}

	return rows.(*sqlx.Rows), nil
}

func CheckInternalAthenaError(_ any, err error) exec.ErrorType {
	if IsInternalAthenaError(err) {
		return exec.ErrorTypeRetryable
	}

	return exec.ErrorTypeUnknown
}

func IsInternalAthenaError(err error) bool {
	errStr := err.Error()

	return strings.Contains(errStr, "Amazon Athena experienced an internal error while executing this query. Please try submitting the query again")
}
