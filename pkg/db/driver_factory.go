package db

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4/database"
)

type DriverFactory interface {
	GetDSN(settings Settings) string
	GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error)
}

var connectionFactories = map[string]DriverFactory{}

func GetDriverFactory(driverName string) (DriverFactory, error) {
	factory, exists := connectionFactories[driverName]

	if !exists {
		return nil, fmt.Errorf("no driver factory defined for %s", driverName)
	}

	return factory, nil
}
