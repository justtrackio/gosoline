package db

import (
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"strings"
)

func runMigrations(logger mon.Logger, db *sqlx.DB, settings *Settings) {
	if !settings.AutoMigrate {
		return
	}

	migrationsTable := "schema_migrations"

	if settings.PrefixedTables {
		application := strings.ToLower(settings.Application)
		application = strings.Replace(application, "-", "_", -1)
		migrationsTable = fmt.Sprintf("%s_schema_migrations", application)
	}

	driver, err := mysql.WithInstance(db.DB, &mysql.Config{
		MigrationsTable: migrationsTable,
	})

	if err != nil {
		logger.Panic(err, "could not initialize driver for db migrations")
	}

	m, err := migrate.NewWithDatabaseInstance(settings.MigrationsPath, settings.DriverName, driver)

	if err != nil {
		logger.Panic(err, "could not initialize migrator for db migrations")
	}

	err = m.Up()

	if err == migrate.ErrNoChange {
		logger.Info("no db migrations to run")
		return
	}

	if err != nil {
		logger.Panic(err, "could not run db migrations")
	}
}
