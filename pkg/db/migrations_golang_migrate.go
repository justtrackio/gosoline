package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/justtrackio/gosoline/pkg/log"
)

func runMigrationGolangMigrate(logger log.Logger, settings *Settings, db *sql.DB) error {
	if !settings.Migrations.Enabled || settings.Migrations.Path == "" {
		return nil
	}

	driverFactory, err := GetDriver(logger, settings.Driver)
	if err != nil {
		return fmt.Errorf("could not get driver factory for %s: %w", settings.Driver, err)
	}

	migrationsTable := "schema_migrations"

	if settings.Migrations.PrefixedTables {
		application := strings.ToLower(settings.Migrations.Application)
		application = strings.Replace(application, "-", "_", -1)
		migrationsTable = fmt.Sprintf("%s_schema_migrations", application)
	}

	driver, err := driverFactory.GetMigrationDriver(db, settings.Uri.Database, migrationsTable)
	if err != nil {
		return fmt.Errorf("could not get migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(settings.Migrations.Path, settings.Driver, driver)
	if err != nil {
		return fmt.Errorf("could not initialize migrator for db migrations: %w", err)
	}

	err = m.Up()

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("no db migrations to run")
		return nil
	}

	if err != nil {
		return fmt.Errorf("could not run db migrations: %w", err)
	}

	return nil
}
