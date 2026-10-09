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

func GetMetricSettings(config cfg.Config) (*Settings, error) {
	var settings Settings
	var err error

	if settings, err = cfg.GetSettings[Settings](config, "metric"); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metric settings: %w", err)
	}

	return &settings, nil
}

func getMetricWriterSettings[T any](config cfg.Config, key string) (*T, error) {
	var settings T
	var err error

	key = fmt.Sprintf("metric.writer_settings.%s", key)

	if settings, err = cfg.GetSettings[T](config, key); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metric writer settings for key '%s': %w", key, err)
	}

	return &settings, nil
}
