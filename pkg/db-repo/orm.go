package db_repo

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jinzhu/gorm"
	"strings"
)

type OrmSettings struct {
	prefixed    bool
	application string
}

func NewOrm(config cfg.Config, logger mon.Logger) *gorm.DB {
	dbClient := db.NewClientWithDefault(logger)

	prefixed := config.GetBool("db_table_prefixed")
	application := config.GetString("app_name")

	application = strings.ToLower(application)
	application = strings.Replace(application, "-", "_", -1)

	return NewOrmWithInterfaces(logger, dbClient, OrmSettings{
		prefixed:    prefixed,
		application: application,
	})
}

func NewOrmWithInterfaces(logger mon.Logger, dbClient gorm.SQLCommon, settings OrmSettings) *gorm.DB {
	orm, err := gorm.Open("mysql", dbClient)

	if err != nil {
		logger.Panic(err, "could not create orm")
	}

	orm.LogMode(false)
	orm = orm.Set("gorm:auto_preload", true)
	orm = orm.Set("gorm:save_associations", false)

	if !settings.prefixed {
		return orm
	}

	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return fmt.Sprintf("%s_%s", settings.application, defaultTableName)
	}

	return orm
}
