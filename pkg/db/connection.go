package db

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"net/url"
	"strconv"
	"sync"
	"time"
)

type Settings struct {
	Application        string        `cfg:"app_name"`
	DriverName         string        `cfg:"db_drivername"`
	Host               string        `cfg:"db_hostname"`
	Port               int           `cfg:"db_port"`
	Database           string        `cfg:"db_database"`
	User               string        `cfg:"db_username"`
	Password           string        `cfg:"db_password"`
	RetryDelay         time.Duration `cfg:"db_retry_wait"`
	ConnectionLifetime time.Duration `cfg:"db_max_connection_lifetime"`
	ParseTime          bool          `cfg:"db_parse_time"`
	AutoMigrate        bool          `cfg:"db_auto_migrate"`
	MigrationsPath     string        `cfg:"db_migrations_path"`
	PrefixedTables     bool          `cfg:"db_table_prefixed"`
}

var defaultConnection = struct {
	lck      sync.Mutex
	instance *sqlx.DB
	err      error
}{}

func ProvideDefaultConnection(config cfg.Config, logger mon.Logger) (*sqlx.DB, error) {
	defaultConnection.lck.Lock()
	defer defaultConnection.lck.Unlock()

	if defaultConnection.err != nil {
		return nil, defaultConnection.err
	}

	if defaultConnection.instance != nil {
		return defaultConnection.instance, nil
	}

	defaultConnection.instance, defaultConnection.err = NewConnection(config, logger)

	return defaultConnection.instance, defaultConnection.err
}

type Connection struct {
	settings *Settings
	db       *sqlx.DB
}

func NewConnection(config cfg.Config, logger mon.Logger) (*sqlx.DB, error) {
	settings := &Settings{}
	config.Unmarshal(settings)

	connection, err := NewConnectionWithInterfaces(settings)

	if err != nil {
		return nil, err
	}

	runMigrations(logger, connection, settings)

	return connection, nil
}

func NewConnectionWithInterfaces(settings *Settings) (*sqlx.DB, error) {
	var err error

	dsn := url.URL{
		User: url.UserPassword(settings.User, settings.Password),
		Host: fmt.Sprintf("tcp(%s:%v)", settings.Host, settings.Port),
		Path: settings.Database,
	}

	qry := dsn.Query()
	qry.Set("multiStatements", "true")
	qry.Set("parseTime", strconv.FormatBool(settings.ParseTime))
	qry.Set("charset", "utf8mb4")
	dsn.RawQuery = qry.Encode()

	db, err := sqlx.Open(settings.DriverName, dsn.String()[2:])

	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(settings.ConnectionLifetime * time.Second)

	return db, nil
}

func GenerateVersionedTableName(table string, version int) string {
	return fmt.Sprintf("V%v_%v", version, table)
}
