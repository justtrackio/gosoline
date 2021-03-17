package db_repo

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jinzhu/gorm"
	"strings"
)

type OrmMigrationSetting struct {
	TablePrefixed bool `cfg:"table_prefixed" default:"true"`
}

type OrmSettings struct {
	Migrations  OrmMigrationSetting `cfg:"migrations"`
	Driver      string              `cfg:"driver" validation:"required"`
	Application string              `cfg:"application" default:"{app_name}"`
}

func NewOrm(config cfg.Config, logger mon.Logger) (*gorm.DB, error) {
	dbClient, err := db.NewClient(config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create dbClient: %w", err)
	}

	settings := OrmSettings{}
	config.UnmarshalKey("db.default", &settings)

	application := settings.Application

	application = strings.ToLower(application)
	application = strings.Replace(application, "-", "_", -1)

	settings.Application = application

	return NewOrmWithInterfaces(logger, dbClient, settings)
}

func NewOrmWithInterfaces(logger mon.Logger, dbClient gorm.SQLCommon, settings OrmSettings) (*gorm.DB, error) {
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

	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return fmt.Sprintf("%s_%s", settings.Application, defaultTableName)
	}

	return orm, nil
}
