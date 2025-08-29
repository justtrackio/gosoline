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
	"github.com/justtrackio/gosoline/pkg/reslife"
)

type connectionCtxKey string

func ProvideConnection(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*sqlx.DB, error) {
	var err error
	var settings *Settings

	if settings, err = ReadSettings(config, name); err != nil {
		return nil, err
	}

	return ProvideConnectionFromSettings(ctx, logger, name, settings)
}

func NewConnection(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*sqlx.DB, error) {
	var err error
	var con *sqlx.DB
	var settings *Settings

	if settings, err = ReadSettings(config, name); err != nil {
		return nil, err
	}

	if con, err = NewConnectionFromSettings(ctx, logger, name, settings); err != nil {
		return nil, err
	}

	return con, nil
}

func ProvideConnectionFromSettings(ctx context.Context, logger log.Logger, name string, settings *Settings) (*sqlx.DB, error) {
	return appctx.Provide(ctx, connectionCtxKey(fmt.Sprint(settings)), func() (*sqlx.DB, error) {
		return NewConnectionFromSettings(ctx, logger, name, settings)
	})
}

func NewConnectionFromSettings(ctx context.Context, logger log.Logger, name string, settings *Settings) (*sqlx.DB, error) {
	var err error
	var connection *sqlx.DB

	if connection, err = NewConnectionWithInterfaces(logger, settings); err != nil {
		return nil, fmt.Errorf("can not create connection: %w", err)
	}

	if err := reslife.AddLifeCycleer(ctx, NewLifecycleManager(name, settings)); err != nil {
		return nil, err
	}

	if err = runMigrations(ctx, logger, settings, connection.DB); err != nil {
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
