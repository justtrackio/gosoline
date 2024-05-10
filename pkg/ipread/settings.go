package ipread

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type RefreshSettings struct {
	Enabled  bool          `cfg:"enabled" default:"false"`
	Interval time.Duration `cfg:"interval" default:"24h"`
}

type ReaderSettings struct {
	Provider string          `cfg:"provider" default:"maxmind"`
	Refresh  RefreshSettings `cfg:"refresh"`
}

func readSettings(config cfg.Config, name string) (*ReaderSettings, error) {
	key := fmt.Sprintf("ipread.%s", name)
	settings := &ReaderSettings{}
	if err := config.UnmarshalKey(key, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ipread settings for %s: %w", name, err)
	}

	return settings, nil
}

func readAllSettings(config cfg.Config) (map[string]*ReaderSettings, error) {
	readerSettings := make(map[string]*ReaderSettings)
	readerMap := config.GetStringMap("ipread", map[string]any{})

	for name := range readerMap {
		settings, err := readSettings(config, name)
		if err != nil {
			return nil, err
		}
		readerSettings[name] = settings
	}

	return readerSettings, nil
}
