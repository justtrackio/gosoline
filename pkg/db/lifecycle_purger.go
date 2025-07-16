package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

var tableExcludes = []string{
	"goose_db_version",
}

type LifeCyclePurger struct {
	logger   log.Logger
	db       *sqlx.DB
	settings *Settings
}

func NewLifeCyclePurger(config cfg.Config, logger log.Logger, connectionName string) (*LifeCyclePurger, error) {
	var err error
	var settings *Settings

	if settings, err = ReadSettings(config, connectionName); err != nil {
		return nil, fmt.Errorf("error reading db settings for connection %q: %w", connectionName, err)
	}

	return NewLifeCyclePurgerWithSettings(logger, settings)
}

func NewLifeCyclePurgerWithSettings(logger log.Logger, settings *Settings) (*LifeCyclePurger, error) {
	var err error
	var db *sqlx.DB

	fkSettings := *settings
	fkSettings.Parameters = map[string]string{
		"FOREIGN_KEY_CHECKS": "0",
	}
	for k, v := range settings.Parameters {
		fkSettings.Parameters[k] = v
	}

	if db, err = NewConnectionWithInterfaces(logger, &fkSettings); err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	return &LifeCyclePurger{
		logger:   logger,
		db:       db,
		settings: settings,
	}, nil
}

func (p LifeCyclePurger) Purge(ctx context.Context) (err error) {
	var tables []string

	defer func() {
		if closeErr := p.db.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("could not close database: %w", closeErr))
		}
	}()

	rows, err := p.db.QueryContext(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?;", p.settings.Uri.Database)
	if err != nil {
		return fmt.Errorf("failed to check tables of database: %w", err)
	}

	for rows.Next() {
		var table string
		if err = rows.Scan(&table); err != nil {
			// on error, we will end the iteration and read the error afterwards with rows.Err()
			break
		}
		tables = append(tables, table)
	}

	if err = rows.Close(); err != nil {
		return fmt.Errorf("could not close rows: %w", err)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("could not iterate over rows: %w", err)
	}

	tables = funk.Filter(tables, func(s string) bool {
		return !funk.Contains(tableExcludes, s)
	})

	if len(tables) == 0 {
		return nil
	}

	chunks := funk.Chunk(tables, int(math.Ceil(float64(len(tables))/float64(runtime.NumCPU()))))

	cfn := coffin.New(ctx)
	for i, chunk := range chunks {
		cfn.GoWithContext(fmt.Sprintf("purge chunk %d", i), func(ctx context.Context) error {
			var table string
			var sqls []string

			for _, table = range chunk {
				sqls = append(sqls, fmt.Sprintf("TRUNCATE TABLE %s;", table))
			}

			if _, err = p.db.ExecContext(ctx, strings.Join(sqls, " ")); err != nil {
				return fmt.Errorf("could not truncate tables: %w", err)
			}

			return nil
		})
	}

	if err = cfn.Wait(); err != nil {
		return fmt.Errorf("error while truncating tables: %w", err)
	}

	return nil
}
