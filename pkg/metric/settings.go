package metric

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type Settings struct {
	Enabled  bool          `cfg:"enabled" default:"false"`
	Interval time.Duration `cfg:"interval" default:"60s"`
	Writers  []string      `cfg:"writers"`
}

func getMetricSettings(config cfg.Config) *Settings {
	settings := &Settings{}
	config.UnmarshalKey("metric", settings)

	return settings
}

func getMetricWriterSettings(config cfg.Config, key string, settings any) {
	key = fmt.Sprintf("metric.writer_settings.%s", key)
	config.UnmarshalKey(key, settings)
}
