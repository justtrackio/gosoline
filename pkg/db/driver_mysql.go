package db

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4/database"
	migrateMysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/justtrackio/gosoline/pkg/log"
)

const DriverMysql = "mysql"

func init() {
	connectionFactories[DriverMysql] = NewMysqlDriver
}

func NewMysqlDriver(logger log.Logger) (Driver, error) {
	sqlLogger := &mysqlLogger{logger: logger}

	if err := mysql.SetLogger(sqlLogger); err != nil {
		return nil, fmt.Errorf("failed to set mysql logger: %w", err)
	}

	return &mysqlDriver{}, nil
}

type mysqlDriver struct{}

func (m *mysqlDriver) GetDSN(settings Settings) string {
	params := map[string]string{
		"charset":      settings.Charset,
		"readTimeout":  settings.Timeouts.ReadTimeout.String(),
		"writeTimeout": settings.Timeouts.WriteTimeout.String(),
	}

	if settings.Timeouts.Timeout > 0 {
		params["timeout"] = settings.Timeouts.Timeout.String()
	}

	cfg := mysql.NewConfig()
	cfg.User = settings.Uri.User
	cfg.Passwd = settings.Uri.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(settings.Uri.Host, strconv.Itoa(settings.Uri.Port))
	cfg.DBName = settings.Uri.Database
	cfg.MultiStatements = settings.MultiStatements
	cfg.ParseTime = settings.ParseTime
	cfg.Params = params
	cfg.Collation = settings.Collation

	return cfg.FormatDSN()
}

func (m *mysqlDriver) GetMigrationDriver(db *sql.DB, database string, migrationsTable string) (database.Driver, error) {
	return migrateMysql.WithInstance(db, &migrateMysql.Config{
		DatabaseName:    database,
		MigrationsTable: migrationsTable,
	})
}

type mysqlLogger struct {
	logger log.Logger
}

func (m mysqlLogger) Print(v ...interface{}) {
	msg := fmt.Sprint(v...)
	m.logger.Error(msg)
}
