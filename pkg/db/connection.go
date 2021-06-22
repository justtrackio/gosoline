package db

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/jmoiron/sqlx"
	"sync"
	"time"
)

type Uri struct {
	Host     string `cfg:"host" validation:"required"`
	Port     int    `cfg:"port" validation:"required"`
	User     string `cfg:"user" validation:"required"`
	Password string `cfg:"password" validation:"required"`
	Database string `cfg:"database" validation:"required"`
}

type Settings struct {
	ConnectionMaxLifetime time.Duration `cfg:"connection_max_lifetime" default:"120s"`
	Driver                string        `cfg:"driver"`
	MaxIdleConnections    int           `cfg:"max_idle_connections" default:"2"` // 0 or negative number=no idle connections, sql driver default=2
	MaxOpenConnections    int           `cfg:"max_open_connections" default:"0"` // 0 or negative number=unlimited, sql driver default=0
	ParseTime             bool          `cfg:"parse_time" default:"true"`

	Uri        Uri               `cfg:"uri"`
	Migrations MigrationSettings `cfg:"migrations"`
}

var defaultConnections = struct {
	lck       sync.Mutex
	instances map[Settings]*sqlx.DB
	errors    map[Settings]error
}{
	instances: make(map[Settings]*sqlx.DB),
	errors:    make(map[Settings]error),
}

func ProvideConnection(config cfg.Config, logger log.Logger, configKey string) (*sqlx.DB, error) {
	defaultConnections.lck.Lock()
	defer defaultConnections.lck.Unlock()

	key := createSettings(config, configKey)

	if err := defaultConnections.errors[key]; err != nil {
		return nil, err
	}

	if instance := defaultConnections.instances[key]; instance != nil {
		return instance, nil
	}

	instance, err := NewConnectionFromSettings(logger, key)

	defaultConnections.instances[key] = instance
	defaultConnections.errors[key] = err

	return defaultConnections.instances[key], defaultConnections.errors[key]
}

func NewConnectionFromSettings(logger log.Logger, settings Settings) (*sqlx.DB, error) {
	var err error
	var connection *sqlx.DB

	if connection, err = NewConnectionWithInterfaces(settings); err != nil {
		return nil, fmt.Errorf("can not create connection: %w", err)
	}

	if err = runMigrations(logger, settings, connection); err != nil {
		return nil, fmt.Errorf("can not run migrations: %w", err)
	}

	publishConnectionMetrics(connection)

	return connection, nil
}

func NewConnectionWithInterfaces(settings Settings) (*sqlx.DB, error) {
	driverFactory, err := GetDriverFactory(settings.Driver)
	if err != nil {
		return nil, fmt.Errorf("could not get dsn provider for driver %s", settings.Driver)
	}

	dsn := driverFactory.GetDSN(settings)

	genDriver, err := getGenericDriver(settings.Driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("could not get driver from %s connection factory: %w", settings.Driver, err)
	}

	metricDriverId := newMetricDriver(genDriver)

	db, err := sqlx.Connect(metricDriverId, dsn)

	if err != nil {
		return nil, fmt.Errorf("can not connect: %w", err)
	}

	db.SetConnMaxLifetime(settings.ConnectionMaxLifetime)
	db.SetMaxIdleConns(settings.MaxIdleConnections)
	db.SetMaxOpenConns(settings.MaxOpenConnections)

	return db, nil
}

func createSettings(config cfg.Config, key string) Settings {
	settings := Settings{
		Migrations: MigrationSettings{},
	}

	config.UnmarshalKey(fmt.Sprintf("db.%s", key), &settings)

	return settings
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
