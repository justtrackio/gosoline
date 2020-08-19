package db

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"net/url"
	"strconv"
)

const DriverMysql = "mysql"

func init() {
	connectionFactories[DriverMysql] = NewMysqlDriverFactory()
}

func NewMysqlDriverFactory() DriverFactory {
	return &mysqlDriverFactory{}
}

type mysqlDriverFactory struct{}

func (m *mysqlDriverFactory) GetDSN(settings Settings) string {
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

func (m *mysqlDriverFactory) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return mysql.WithInstance(db, &mysql.Config{
		DatabaseName:    database,
		MigrationsTable: migrationsTable,
	})
}
