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
	key := GetLeaderElectionConfigKey(name)
	defaultKey := GetLeaderElectionConfigKey("default")

	settings := &DdbLeaderElectionSettings{}
	if err := config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultsFromKey(defaultKey, ".")); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ddb leader election settings: %w", err)
	}

	return settings, nil
}
