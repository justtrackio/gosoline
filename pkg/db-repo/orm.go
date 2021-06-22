package db_repo

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/log"
	"github.com/iancoleman/strcase"
	"github.com/jinzhu/gorm"
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

	return NewOrmWithInterfaces(dbClient, settings)
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

	return NewOrmWithInterfaces(dbClient, ormSettings)
}

func NewOrmWithInterfaces(dbClient gorm.SQLCommon, settings OrmSettings) (*gorm.DB, error) {
	orm, err := gorm.Open(settings.Driver, dbClient)

	if err != nil {
		return nil, fmt.Errorf("could not create gorm: %w", err)
	}

	orm.LogMode(false)
	orm = orm.Set("gorm:auto_preload", true)
	orm = orm.Set("gorm:save_associations", false)

	if !settings.Migrations.TablePrefixed {
		return orm, nil
	}

	prefix := strcase.ToSnake(settings.Application)

	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return fmt.Sprintf("%s_%s", prefix, defaultTableName)
	}

	return orm, nil
}
