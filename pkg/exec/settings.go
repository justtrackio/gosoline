package exec

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type BackoffSettings struct {
	CancelDelay     time.Duration `cfg:"cancel_delay"`
	InitialInterval time.Duration `cfg:"initial_interval" default:"50ms"`
	MaxAttempts     int           `cfg:"max_attempts" default:"10"`
	MaxElapsedTime  time.Duration `cfg:"max_elapsed_time" default:"10m"`
	MaxInterval     time.Duration `cfg:"max_interval" default:"10s"`
}

func ReadBackoffSettings(config cfg.Config, paths ...string) (BackoffSettings, error) {
	typ := "default"
	paths = append(paths, "exec")

	for i := len(paths) - 1; i >= 0; i-- {
		key := fmt.Sprintf("%s.backoff", paths[i])
		keyType := fmt.Sprintf("%s.backoff.type", paths[i])

		if !config.IsSet(key) {
			continue
		}

		if !config.IsSet(keyType) {
			typ = "custom"

			continue
		}

		var err error
		typ, err = config.GetString(keyType)
		if err != nil {
			return BackoffSettings{}, fmt.Errorf("could not get backoff type: %w", err)
		}
	}

	if settings, ok := predefined[typ]; ok {
		return settings, nil
	}

	additionalDefaults := make([]cfg.UnmarshalDefaults, 0)

	for i := 1; i < len(paths); i++ {
		key := fmt.Sprintf("%s.backoff", paths[i])
		additionalDefaults = append(additionalDefaults, cfg.UnmarshalWithDefaultsFromKey(key, "."))
	}

	key := fmt.Sprintf("%s.backoff", paths[0])
	settings := &BackoffSettings{}
	if err := config.UnmarshalKey(key, settings, additionalDefaults...); err != nil {
		return BackoffSettings{}, fmt.Errorf("failed to unmarshal backoff settings for key %s: %w", key, err)
	}

	return *settings, nil
}

var predefined = map[string]BackoffSettings{
	"api": {
		InitialInterval: time.Millisecond * 100,
		MaxElapsedTime:  time.Second * 10,
		MaxInterval:     time.Second,
	},
	"once": {
		MaxAttempts:    1,
		MaxElapsedTime: 0,
	},
	"infinite": {
		InitialInterval: time.Millisecond * 50,
		MaxAttempts:     0,
		MaxElapsedTime:  0,
		MaxInterval:     time.Second * 10,
	},
}
