package db

import (
	"fmt"
	"github.com/applike/gosoline/pkg/log"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"strings"
)

type MigrationSettings struct {
	Application    string `cfg:"application" default:"{app_name}"`
	Path           string `cfg:"path"`
	PrefixedTables bool   `cfg:"prefixed_tables" default:"false"`
	Enabled        bool   `cfg:"enabled" default:"false"`
}

func runMigrations(logger log.Logger, settings Settings, db *sqlx.DB) error {
	if !settings.Migrations.Enabled || settings.Migrations.Path == "" {
		return nil
	}

	driverFactory, err := GetDriverFactory(settings.Driver)

	if err != nil {
		return fmt.Errorf("could not get driver factory for %s: %w", settings.Driver, err)
	}

	migrationsTable := "schema_migrations"

	if settings.Migrations.PrefixedTables {
		application := strings.ToLower(settings.Migrations.Application)
		application = strings.Replace(application, "-", "_", -1)
		migrationsTable = fmt.Sprintf("%s_schema_migrations", application)
	}

	driver, err := driverFactory.GetMigrationDriver(db.DB, settings.Uri.Database, migrationsTable)

	if err != nil {
		return fmt.Errorf("could not get migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(settings.Migrations.Path, settings.Driver, driver)

	if err != nil {
		return fmt.Errorf("could not initialize migrator for db migrations: %w", err)
	}

	err = m.Up()

	if err == migrate.ErrNoChange {
		logger.Info("no db migrations to run")
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not run db migrations: %w", err)
	}

	return nil
}
