package db

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/cockroachdb"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

const DriverNamePostgres = "postgres"

func init() {
	connectionFactories[DriverNamePostgres] = NewPostgresDriverFactory()
}

func NewPostgresDriverFactory() DriverFactory {
	return &postgresDriverFactory{}
}

type postgresDriverFactory struct{}

func (m *postgresDriverFactory) GetDSN(settings Settings) string {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(settings.Uri.User, settings.Uri.Password),
		Host:   fmt.Sprintf("%s:%d", settings.Uri.Host, settings.Uri.Port),
	}

	qry := dsn.Query()
	qry.Set("dbname", settings.Uri.Database)

	if !settings.SslModeEnabled {
		qry.Set("sslmode", "disable")
	}

	dsn.RawQuery = qry.Encode()

	return dsn.String()
}

func (m *postgresDriverFactory) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return cockroachdb.WithInstance(db, &cockroachdb.Config{
		MigrationsTable: migrationsTable,
		DatabaseName:    database,
	})
}
