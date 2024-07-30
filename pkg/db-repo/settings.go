package db_repo

import "github.com/justtrackio/gosoline/pkg/cfg"

type changeHistoryManagerSettings struct {
	TableSuffix string `cfg:"table_suffix" default:"history"`
}

type OrmMigrationSetting struct {
	TablePrefixed bool `cfg:"table_prefixed" default:"true"`
}

type OrmSettings struct {
	Migrations  OrmMigrationSetting `cfg:"migrations"`
	Driver      string              `cfg:"driver"      validation:"required"`
	Application string              `cfg:"application"                       default:"{app_name}"`
}

type Settings struct {
	cfg.AppId
	Metadata Metadata
}
