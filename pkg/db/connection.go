package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

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
	driver, err := GetDriver(logger, settings.Driver)
	if err != nil {
		return nil, fmt.Errorf("could not get dsn provider for driver %s", settings.Driver)
	}

	dsn := driver.GetDSN(settings)

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
