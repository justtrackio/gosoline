package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
)

type MigrationProvider func(logger log.Logger, settings *Settings, db *sql.DB) error

func AddMigrationProvider(name string, provider MigrationProvider) {
	migrationProviders[name] = provider
}

var migrationProviders = map[string]MigrationProvider{
	"golang-migrate": runMigrationGolangMigrate,
	"goose":          runMigrationGoose,
}

func runMigrations(logger log.Logger, settings *Settings, db *sql.DB) error {
	logger = logger.WithChannel("db-migrations")

	if !settings.Migrations.Enabled || settings.Migrations.Path == "" {
		logger.Info("migrations not enabled")
		return nil
	}

	if settings.Migrations.Path == "" {
		logger.Info("migrations enabled but no path provided")
		return nil
	}

	var ok bool
	var err error
	var provider MigrationProvider

	if provider, ok = migrationProviders[settings.Migrations.Provider]; !ok {
		return fmt.Errorf("there is no migration provider of type %s available", settings.Migrations.Provider)
	}

	start := time.Now()

	if err = provider(logger, settings, db); err != nil {
		return fmt.Errorf("running migration provider %s failed: %w", settings.Migrations.Provider, err)
	}

	logger.Info("migrated db in %s", time.Since(start))

	return nil
}
