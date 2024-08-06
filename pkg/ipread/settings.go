package ipread

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type RefreshSettings struct {
	Enabled  bool          `cfg:"enabled"  default:"false"`
	Interval time.Duration `cfg:"interval" default:"24h"`
}

type ReaderSettings struct {
	Provider string          `cfg:"provider" default:"maxmind"`
	Refresh  RefreshSettings `cfg:"refresh"`
}

func readSettings(config cfg.Config, name string) *ReaderSettings {
	key := fmt.Sprintf("ipread.%s", name)
	settings := &ReaderSettings{}
	config.UnmarshalKey(key, settings)

	return settings
}

func readAllSettings(config cfg.Config) map[string]*ReaderSettings {
	readerSettings := make(map[string]*ReaderSettings)
	readerMap := config.GetStringMap("ipread", map[string]interface{}{})

	for name := range readerMap {
		settings := readSettings(config, name)
		readerSettings[name] = settings
	}

	return readerSettings
}

type MaxmindSettings struct {
	Database     string `cfg:"database"`
	S3ClientName string `cfg:"s3_client_name"`
}
