package db_repo

import (
	"fmt"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type OrmMigrationSetting struct {
	TablePrefixed bool `cfg:"table_prefixed" default:"true"`
}

type OrmSettings struct {
	Migrations  OrmMigrationSetting `cfg:"migrations"`
	Driver      string              `cfg:"driver" validation:"required"`
	Application string              `cfg:"application" default:"{app_name}"`
}

func NewOrm(config cfg.Config, logger log.Logger) (*gorm.DB, error) {
	dbClient, err := db.NewClient(config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create dbClient: %w", err)
	}

	settings := OrmSettings{}
	config.UnmarshalKey("db.default", &settings)

	return NewOrmWithInterfaces(clock.Provider, dbClient, settings)
}

func NewOrmWithDbSettings(logger log.Logger, dbSettings db.Settings, application string) (*gorm.DB, error) {
	dbClient, err := db.NewClientWithSettings(logger, dbSettings)
	if err != nil {
		return nil, fmt.Errorf("can not create dbClient: %w", err)
	}

	ormSettings := OrmSettings{
		Migrations: OrmMigrationSetting{
			TablePrefixed: dbSettings.Migrations.PrefixedTables,
		},
		Driver:      dbSettings.Driver,
		Application: application,
	}

	return NewOrmWithInterfaces(clock.Provider, dbClient, ormSettings)
}

func NewOrmWithInterfaces(clock clock.Clock, dbClient gorm.ConnPool, settings OrmSettings) (*gorm.DB, error) {
	logger := noopLogger{}

	namingStrategy := schema.NamingStrategy{}

	if settings.Migrations.TablePrefixed {
		namingStrategy.TablePrefix = strcase.ToSnake(settings.Application)
	}

	config := gorm.Config{
		NamingStrategy: namingStrategy,
		Logger:         &logger,
		NowFunc: func() time.Time {
			return clock.Now()
		},
	}

	dial, err := newDialector(settings.Driver, dbClient)
	if err != nil {
		return nil, err
	}

	orm, err := gorm.Open(dial, &config)
	if err != nil {
		return nil, fmt.Errorf("could not create gorm: %w", err)
	}

	return orm, nil
}

func newDialector(driver string, client gorm.ConnPool) (gorm.Dialector, error) {
	switch driver {
	case "mysql":
		return mysql.New(mysql.Config{
			Conn:                     client,
			DisableDatetimePrecision: true,
			DefaultStringSize:        255,
		}), nil
	}

	return nil, fmt.Errorf("no dialector defined for %s", driver)
}
