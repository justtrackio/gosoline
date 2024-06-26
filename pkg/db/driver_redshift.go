package db

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4/database"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/redshift"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/lib/pq"
)

const DriverNameRedshift = "redshift"

func init() {
	sql.Register(DriverNameRedshift, &pq.Driver{})
	connectionFactories[DriverNameRedshift] = NewRedshiftDriver
}

func NewRedshiftDriver(logger log.Logger) (Driver, error) {
	return &redshiftDriver{}, nil
}

type redshiftDriver struct{}

func (m *redshiftDriver) GetDSN(settings *Settings) string {
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

func (m *redshiftDriver) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return redshift.WithInstance(db, &redshift.Config{
		MigrationsTable: migrationsTable,
		DatabaseName:    database,
	})
}
