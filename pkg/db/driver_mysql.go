package db

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4/database"
	migrateMysql "github.com/golang-migrate/migrate/v4/database/mysql"
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
	cfg := mysql.NewConfig()
	cfg.User = settings.Uri.User
	cfg.Passwd = settings.Uri.Password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", settings.Uri.Host, settings.Uri.Port)
	cfg.DBName = settings.Uri.Database
	cfg.MultiStatements = true
	cfg.ParseTime = true
	cfg.Collation = "utf8mb4_general_ci"

	return cfg.FormatDSN()
}

func (m *mysqlDriverFactory) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return migrateMysql.WithInstance(db, &migrateMysql.Config{
		DatabaseName:    database,
		MigrationsTable: migrationsTable,
	})
}
