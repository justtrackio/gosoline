package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
)

type MigrationSettings struct {
	Application    string `cfg:"application" default:"{app.name}"`
	Enabled        bool   `cfg:"enabled" default:"false"`
	Reset          bool   `cfg:"reset" default:"false"`
	Path           string `cfg:"path"`
	PrefixedTables bool   `cfg:"prefixed_tables" default:"false"`
	Provider       string `cfg:"provider" default:"goose"`
}

type MigrationProvider func(ctx context.Context, logger log.Logger, settings *Settings, db *sql.DB) error

func AddMigrationProvider(name string, provider MigrationProvider) {
	migrationProviders[name] = provider
}

var migrationProviders = map[string]MigrationProvider{
	"golang-migrate": runMigrationGolangMigrate,
	"goose":          runMigrationGoose,
}

func runMigrations(ctx context.Context, logger log.Logger, settings *Settings, _ *sql.DB) error {
	logger = logger.WithChannel("db-migrations")

	if !settings.Migrations.Enabled || settings.Migrations.Path == "" {
		logger.Info(ctx, "migrations not enabled")

		return nil
	}

	if settings.Migrations.Path == "" {
		logger.Info(ctx, "migrations enabled but no path provided")

		return nil
	}

	var ok bool
	var err error
	var provider MigrationProvider

	if provider, ok = migrationProviders[settings.Migrations.Provider]; !ok {
		return fmt.Errorf("there is no migration provider of type %s available", settings.Migrations.Provider)
	}

	// Migrations may contain multi-statement SQL files; open a dedicated
	// connection with MultiStatements enabled. Runtime connections keep the
	// secure default (false) to prevent second-order SQL injection.
	migrationDB, err := openMigrationDB(logger, settings)
	if err != nil {
		return fmt.Errorf("can not open migration connection: %w", err)
	}
	defer migrationDB.Close()

	start := time.Now()

	if err = resetMigrations(ctx, logger, settings, migrationDB); err != nil {
		return fmt.Errorf("could not reset migrations: %w", err)
	}

	if err = provider(ctx, logger, settings, migrationDB); err != nil {
		return fmt.Errorf("running migration provider %s failed: %w", settings.Migrations.Provider, err)
	}

	logger.Info(ctx, "migrated db in %s", time.Since(start))

	return nil
}

func openMigrationDB(logger log.Logger, settings *Settings) (*sql.DB, error) {
	drv, err := GetDriver(logger, settings.Driver)
	if err != nil {
		return nil, fmt.Errorf("can not get driver: %w", err)
	}

	migrationSettings := *settings
	migrationSettings.MultiStatements = true

	dsn := drv.GetDSN(&migrationSettings)

	db, err := sql.Open(settings.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("can not open connection: %w", err)
	}

	return db, nil
}

func resetMigrations(ctx context.Context, logger log.Logger, settings *Settings, db *sql.DB) error {
	if !settings.Migrations.Reset {
		return nil
	}

	logger.Info(ctx, "resetting database %s to rerun migrations", settings.Uri.Database)

	sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s", settings.Uri.Database)
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("can not drop database %s: %w", settings.Uri.Database, err)
	}

	sql = fmt.Sprintf("CREATE DATABASE %s", settings.Uri.Database)
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("can not create database %s: %w", settings.Uri.Database, err)
	}

	sql = fmt.Sprintf("USE %s", settings.Uri.Database)
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("can not use database %s: %w", settings.Uri.Database, err)
	}

	return nil
}
