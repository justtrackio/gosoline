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

func getMetricSettings(config cfg.Config) (*Settings, error) {
	settings := &Settings{}
	if err := config.UnmarshalKey("metric", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metric settings: %w", err)
	}

	return settings, nil
}

func getMetricWriterSettings(config cfg.Config, key string, settings any) error {
	key = fmt.Sprintf("metric.writer_settings.%s", key)
	if err := config.UnmarshalKey(key, settings); err != nil {
		return fmt.Errorf("failed to unmarshal metric writer settings for key '%s': %w", key, err)
	}

	return nil
}
