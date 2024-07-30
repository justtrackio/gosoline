package db

import "time"

type Uri struct {
	Host     string `cfg:"host"     default:"localhost" validation:"required"`
	Port     int    `cfg:"port"     default:"3306"      validation:"required"`
	User     string `cfg:"user"                         validation:"required"`
	Password string `cfg:"password"                     validation:"required"`
	Database string `cfg:"database"                     validation:"required"`
}

type Settings struct {
	Charset               string            `cfg:"charset"                 default:"utf8mb4"`
	Collation             string            `cfg:"collation"               default:"utf8mb4_general_ci"`
	ConnectionMaxIdleTime time.Duration     `cfg:"connection_max_idletime" default:"120s"`
	ConnectionMaxLifetime time.Duration     `cfg:"connection_max_lifetime" default:"120s"`
	Driver                string            `cfg:"driver"`
	MaxIdleConnections    int               `cfg:"max_idle_connections"    default:"2"` // 0 or negative number=no idle connections, sql driver default=2
	MaxOpenConnections    int               `cfg:"max_open_connections"    default:"0"` // 0 or negative number=unlimited, sql driver default=0
	Migrations            MigrationSettings `cfg:"migrations"`
	MultiStatements       bool              `cfg:"multi_statements"        default:"true"`
	ParseTime             bool              `cfg:"parse_time"              default:"true"`
	Retry                 SettingsRetry     `cfg:"retry"`
	Timeouts              SettingsTimeout   `cfg:"timeouts"`
	Uri                   Uri               `cfg:"uri"`
}

type SettingsRetry struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

type SettingsTimeout struct {
	ReadTimeout  time.Duration `cfg:"readTimeout"  default:"0"` // I/O read timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
	WriteTimeout time.Duration `cfg:"writeTimeout" default:"0"` // I/O write timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
	Timeout      time.Duration `cfg:"timeout"      default:"0"` // Timeout for establishing connections, aka dial timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
}

type MigrationSettings struct {
	Application    string `cfg:"application"     default:"{app_name}"`
	Enabled        bool   `cfg:"enabled"         default:"false"`
	Path           string `cfg:"path"`
	PrefixedTables bool   `cfg:"prefixed_tables" default:"false"`
	Provider       string `cfg:"provider"        default:"goose"`
}
