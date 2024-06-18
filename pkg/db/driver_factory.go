package db

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/justtrackio/gosoline/pkg/log"
)

type DriverFactory func(logger log.Logger) (Driver, error)

type Driver interface {
	GetDSN(settings Settings) string
	GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error)
}

var connectionFactories = map[string]DriverFactory{}

func GetDriver(logger log.Logger, driverName string) (Driver, error) {
	var ok bool
	var err error
	var factory DriverFactory
	var driver Driver

	if factory, ok = connectionFactories[driverName]; !ok {
		return nil, fmt.Errorf("no driver factory defined for %s", driverName)
	}

	if driver, err = factory(logger); err != nil {
		return nil, fmt.Errorf("failed to get driver for %s: %w", driverName, err)
	}

	return driver, nil
}
