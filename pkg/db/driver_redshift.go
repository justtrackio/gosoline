package db

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4/database"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/redshift"
	"github.com/lib/pq"
	"net/url"
)

const DriverNameRedshift = "redshift"

func init() {
	sql.Register(DriverNameRedshift, &pq.Driver{})
	connectionFactories[DriverNameRedshift] = NewRedshiftDriverFactory()
}

func NewRedshiftDriverFactory() DriverFactory {
	return &redshiftDriverFactory{}
}

type redshiftDriverFactory struct{}

func (m *redshiftDriverFactory) GetDSN(settings Settings) string {
	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(settings.Uri.User, settings.Uri.Password),
		Host:   fmt.Sprintf("%s:%d", settings.Uri.Host, settings.Uri.Port),
	}

	qry := dsn.Query()
	qry.Set("dbname", settings.Uri.Database)
	dsn.RawQuery = qry.Encode()

	return dsn.String()
}

func (m *redshiftDriverFactory) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return redshift.WithInstance(db, &redshift.Config{
		MigrationsTable: migrationsTable,
		DatabaseName:    database,
	})
}
