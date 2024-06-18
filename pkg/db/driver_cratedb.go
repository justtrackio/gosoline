package db

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/justtrackio/gosoline/pkg/log"
)

const DriverNameCrateDb = "cratedb"

func init() {
	sql.Register(DriverNameCrateDb, stdlib.GetDefaultDriver())
	connectionFactories[DriverNameCrateDb] = NewCrateDbDriver
}

type crateDbDriver struct{}

func NewCrateDbDriver(logger log.Logger) (Driver, error) {
	return &crateDbDriver{}, nil
}

func (c crateDbDriver) GetDSN(settings Settings) string {
	dsn := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", settings.Uri.Host, settings.Uri.Port),
		User:   url.UserPassword(settings.Uri.User, settings.Uri.Password),
		Path:   settings.Uri.Database,
	}

	return dsn.String()
}

func (c crateDbDriver) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: migrationsTable,
		DatabaseName:    database,
	})
}
