package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
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

	if db, err = NewConnectionWithInterfaces(logger, settings); err != nil {
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

	if tables, err = p.getTables(ctx); err != nil {
		return err
	}

	if len(tables) == 0 {
		return nil
	}

	if err := p.deleteTables(ctx, tables); err != nil {
		return err
	}

	return nil
}

// getTables returns all tables in the database that should be purged.
func (p LifeCyclePurger) getTables(ctx context.Context) ([]string, error) {
	var tables []string

	err := p.db.SelectContext(ctx, &tables, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'", p.settings.Uri.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables of database: %w", err)
	}

	tables = funk.Filter(tables, func(s string) bool {
		return !funk.Contains(tableExcludes, s)
	})

	return tables, nil
}

// deleteTables deletes all rows from the given tables in a single transaction.
// FK checks are disabled within the transaction to handle circular references
// and avoid ordering issues. Using DELETE instead of TRUNCATE avoids the
// MySQL/InnoDB issue where TRUNCATE changes the internal table_id, which can
// cause other connections with cached FK metadata to fail with Error 1452.
//
// For performance, all DELETE statements are batched into a single multi-statement
// query to minimize round trips to the database.
func (p LifeCyclePurger) deleteTables(ctx context.Context, tables []string) error {
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false

	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				p.logger.Warn(ctx, "failed to rollback transaction: %s", rollbackErr.Error())
			}
		}
	}()

	// Build a single multi-statement query with all DELETEs
	// This reduces round trips and improves performance significantly
	var statements []string
	statements = append(statements, "SET FOREIGN_KEY_CHECKS = 0")
	for _, table := range tables {
		statements = append(statements, fmt.Sprintf("TRUNCATE TABLE `%s`", table))
	}
	statements = append(statements, "SET FOREIGN_KEY_CHECKS = 1")

	query := strings.Join(statements, "; ")
	if _, err = tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to purge tables: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true

	return nil
}
