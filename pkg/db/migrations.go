package db

import (
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"strings"
)

func (c *Connection) runMigrations() {
	if !c.settings.AutoMigrate {
		return
	}

	migrationsTable := "schema_migrations"

	if c.settings.PrefixedTables {
		application := strings.ToLower(c.settings.Application)
		application = strings.Replace(application, "-", "_", -1)
		migrationsTable = fmt.Sprintf("%s_schema_migrations", application)
	}

	driver, err := mysql.WithInstance(c.db.DB, &mysql.Config{
		MigrationsTable: migrationsTable,
	})

	if err != nil {
		c.logger.Panic(err, "could not initialize driver for db migrations")
	}

	m, err := migrate.NewWithDatabaseInstance(c.settings.MigrationsPath, c.settings.DriverName, driver)

	if err != nil {
		c.logger.Panic(err, "could not initialize migrator for db migrations")
	}

	err = m.Up()

	if err == migrate.ErrNoChange {
		c.logger.Info("no db migrations to run")
		return
	}

	if err != nil {
		c.logger.Panic(err, "could not run db migrations")
	}
}
