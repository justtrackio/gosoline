package ddb

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type TableNamingSettings struct {
	TablePattern   string `cfg:"table_pattern,nodecode" default:"{app.namespace}-leader-elections"`
	TableDelimiter string `cfg:"table_delimiter" default:"-"`
}

type DdbLeaderElectionSettings struct {
	Naming        TableNamingSettings `cfg:"naming"`
	ClientName    string              `cfg:"client_name" default:"default"`
	GroupId       string              `cfg:"group_id" default:"{app.name}"`
	LeaseDuration time.Duration       `cfg:"lease_duration" default:"1m"`
}

func ReadLeaderElectionDdbSettings(config cfg.Config, name string) (*DdbLeaderElectionSettings, error) {
	var settings DdbLeaderElectionSettings
	var err error

	key := GetLeaderElectionConfigKey(name)
	defaultKey := GetLeaderElectionConfigKey("default")

	if settings, err = cfg.GetSettings[DdbLeaderElectionSettings](config, key, cfg.UnmarshalWithDefaultsFromKey(defaultKey, ".")); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ddb leader election settings for name %q (key %q, default key %q): %w", name, key, defaultKey, err)
	}

	return &settings, nil
}
