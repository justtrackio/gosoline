package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Uri struct {
	Host     string `cfg:"host" default:"localhost" validation:"required"`
	Port     int    `cfg:"port" default:"3306" validation:"required"`
	User     string `cfg:"user" validation:"required"`
	Password string `cfg:"password" validation:"required"`
	Database string `cfg:"database" validation:"required"`
}

type Settings struct {
	Charset               string            `cfg:"charset" default:"utf8mb4"`
	Collation             string            `cfg:"collation" default:"utf8mb4_general_ci"`
	ConnectionMaxIdleTime time.Duration     `cfg:"connection_max_idletime" default:"120s"`
	ConnectionMaxLifetime time.Duration     `cfg:"connection_max_lifetime" default:"120s"`
	Driver                string            `cfg:"driver"`
	MaxIdleConnections    int               `cfg:"max_idle_connections" default:"2"` // 0 or negative number=no idle connections, sql driver default=2
	MaxOpenConnections    int               `cfg:"max_open_connections" default:"0"` // 0 or negative number=unlimited, sql driver default=0
	Migrations            MigrationSettings `cfg:"migrations"`
	MultiStatements       bool              `cfg:"multi_statements" default:"true"`
	Parameters            map[string]string `cfg:"parameters"`
	ParseTime             bool              `cfg:"parse_time" default:"true"`
	Retry                 SettingsRetry     `cfg:"retry"`
	Timeouts              SettingsTimeout   `cfg:"timeouts"`
	Uri                   Uri               `cfg:"uri"`
}

type SettingsRetry struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

type SettingsTimeout struct {
	ReadTimeout  time.Duration `cfg:"readTimeout" default:"0"`  // I/O read timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
	WriteTimeout time.Duration `cfg:"writeTimeout" default:"0"` // I/O write timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
	Timeout      time.Duration `cfg:"timeout" default:"0"`      // Timeout for establishing connections, aka dial timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
}

type connectionCtxKey string

func ProvideConnection(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*sqlx.DB, error) {
	return appctx.Provide(ctx, connectionCtxKey(name), func() (*sqlx.DB, error) {
		return NewConnection(config, logger, name)
	})
}

func NewConnection(config cfg.Config, logger log.Logger, name string) (*sqlx.DB, error) {
	var err error
	var con *sqlx.DB
	var settings *Settings

	if settings, err = readSettings(config, name); err != nil {
		return nil, err
	}

	if con, err = NewConnectionFromSettings(logger, settings); err != nil {
		return nil, err
	}

	return con, nil
}

func ProvideConnectionFromSettings(ctx context.Context, logger log.Logger, settings *Settings) (*sqlx.DB, error) {
	return appctx.Provide(ctx, connectionCtxKey(fmt.Sprint(settings)), func() (*sqlx.DB, error) {
		return NewConnectionFromSettings(logger, settings)
	})
}

func NewConnectionFromSettings(logger log.Logger, settings *Settings) (*sqlx.DB, error) {
	var err error
	var connection *sqlx.DB

	if connection, err = NewConnectionWithInterfaces(logger, settings); err != nil {
		return nil, fmt.Errorf("can not create connection: %w", err)
	}

	if err = runMigrations(logger, settings, connection.DB); err != nil {
		return nil, fmt.Errorf("can not run migrations: %w", err)
	}

	publishConnectionMetrics(connection)

	return connection, nil
}

func NewConnectionWithInterfaces(logger log.Logger, settings *Settings) (*sqlx.DB, error) {
	drv, err := GetDriver(logger, settings.Driver)
	if err != nil {
		return nil, fmt.Errorf("could not get dsn provider for driver %s", settings.Driver)
	}

	dsn := drv.GetDSN(settings)

	genDriver, err := getGenericDriver(settings.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("could not get driver from %s connection factory: %w", settings.Driver, err)
	}

	metricDriverId := newMetricDriver(genDriver)

	db, err := sqlx.Connect(metricDriverId, dsn)
	if err != nil {
		return nil, fmt.Errorf("can not connect: %w", err)
	}

	db.SetConnMaxIdleTime(settings.ConnectionMaxIdleTime)
	db.SetConnMaxLifetime(settings.ConnectionMaxLifetime)
	db.SetMaxIdleConns(settings.MaxIdleConnections)
	db.SetMaxOpenConns(settings.MaxOpenConnections)

	return db, nil
}

func readSettings(config cfg.Config, name string) (*Settings, error) {
	key := fmt.Sprintf("db.%s", name)

	if !config.IsSet(key) {
		return nil, fmt.Errorf("there is no db connection with name %q configured", name)
	}

	settings := &Settings{}
	config.UnmarshalKey(key, settings)

	return settings, nil
}

func getGenericDriver(driverName, dsn string) (driver.Driver, error) {
	initDb, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	err = initDb.Close()
	if err != nil {
		return nil, err
	}

	genDriver := initDb.Driver()

	return genDriver, nil
}
