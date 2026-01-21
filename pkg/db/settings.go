package db

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

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
	Parameters            map[string]string `cfg:"parameters"`
	ParseTime             bool              `cfg:"parse_time"              default:"true"`
	Retry                 SettingsRetry     `cfg:"retry"`
	Timeouts              SettingsTimeout   `cfg:"timeouts"`
	Uri                   SettingsUri       `cfg:"uri"`
	InterpolateParams     bool              `cfg:"interpolate_params" default:"true"`
}

type SettingsUri struct {
	Host     string `cfg:"host"     default:"localhost" validation:"required"`
	Port     int    `cfg:"port"     default:"3306"      validation:"required"`
	User     string `cfg:"user"                         validation:"required"`
	Password string `cfg:"password"                     validation:"required"`
	Database string `cfg:"database"                     validation:"required"`
}

type SettingsRetry struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

type SettingsTimeout struct {
	ReadTimeout  time.Duration `cfg:"readTimeout"  default:"0"` // I/O read timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
	WriteTimeout time.Duration `cfg:"writeTimeout" default:"0"` // I/O write timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
	Timeout      time.Duration `cfg:"timeout"      default:"0"` // Timeout for establishing connections, aka dial timeout. The value must be a decimal number with a unit suffix ("ms", "s", "m", "h"), such as "30s", "0.5m" or "1m30s".
}

func ReadSettings(config cfg.Config, name string) (*Settings, error) {
	key := fmt.Sprintf("db.%s", name)

	if !config.IsSet(key) {
		return nil, fmt.Errorf("there is no db connection with name %q configured", name)
	}

	settings := &Settings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal db settings for %s: %w", name, err)
	}

	return settings, nil
}
