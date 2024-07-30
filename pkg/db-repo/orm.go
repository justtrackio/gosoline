package db_repo

import (
	"context"
	"fmt"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewOrm(ctx context.Context, config cfg.Config, logger log.Logger) (*gorm.DB, error) {
	connection, err := db.ProvideConnection(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create db connection : %w", err)
	}

	settings := OrmSettings{}
	config.UnmarshalKey("db.default", &settings)

	return NewOrmWithInterfaces(connection, settings)
}

func NewOrmWithDbSettings(logger log.Logger, dbSettings *db.Settings, application string) (*gorm.DB, error) {
	dbClient, err := db.NewConnectionFromSettings(logger, dbSettings)
	if err != nil {
		return nil, fmt.Errorf("can not connect to sql database: %w", err)
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
	orm.SetLogger(&noopLogger{})
	orm = orm.Set("gorm:auto_preload", true)
	orm = orm.Set("gorm:save_associations", false)

	orm.SetNowFuncOverride(func() time.Time {
		return clock.Provider.Now()
	})

	if !settings.Migrations.TablePrefixed {
		return orm, nil
	}

	prefix := strcase.ToSnake(settings.Application)

	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return fmt.Sprintf("%s_%s", prefix, defaultTableName)
	}

	return orm, nil
}
