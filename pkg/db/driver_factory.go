package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/redshift"
	"github.com/lib/pq"
	"net/url"
	"strconv"
)

func init() {
	sql.Register("redshift", &pq.Driver{})
}

type DriverFactory interface {
	GetDSN(settings Settings) string
	GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error)
}

var connectionFactories = map[string]DriverFactory{
	"mysql":    NewMysqlFactory(),
	"redshift": NewRedshiftFactory(),
}

func GetDriverFactory(driverName string) (DriverFactory, error) {
	factory, exists := connectionFactories[driverName]

	if !exists {
		return nil, fmt.Errorf("no driver factory defined for %s", driverName)
	}

	return factory, nil
}

func NewMysqlFactory() DriverFactory {
	return &mysqlFactory{}
}

type mysqlFactory struct{}

func (m *mysqlFactory) GetDSN(settings Settings) string {
	dsn := url.URL{
		User: url.UserPassword(settings.Uri.User, settings.Uri.Password),
		Host: fmt.Sprintf("tcp(%s:%d)", settings.Uri.Host, settings.Uri.Port),
		Path: settings.Uri.Database,
	}

	qry := dsn.Query()
	qry.Set("multiStatements", "true")
	qry.Set("parseTime", strconv.FormatBool(settings.ParseTime))
	qry.Set("charset", "utf8mb4")
	dsn.RawQuery = qry.Encode()

	uri := dsn.String()

	return uri[2:]
}

func (m *mysqlFactory) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return mysql.WithInstance(db, &mysql.Config{
		DatabaseName:    database,
		MigrationsTable: migrationsTable,
	})
}

func NewRedshiftFactory() DriverFactory {
	return &postgresFactory{}
}

type postgresFactory struct{}

func (m *postgresFactory) GetDSN(settings Settings) string {
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

func (m *postgresFactory) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return redshift.WithInstance(db, &redshift.Config{
		MigrationsTable: migrationsTable,
		DatabaseName:    database,
	})
}
