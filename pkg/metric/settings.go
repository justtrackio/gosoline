package metric

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type CloudwatchNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}/{env}/{family}/{group}-{app}"`
}
type CloudWatchSettings struct {
	Naming    CloudwatchNamingSettings `cfg:"naming"`
	Aggregate bool                     `cfg:"aggregate" default:"true"`
}

type PrometheusServerSettings struct {
	Enabled bool            `cfg:"enabled" default:"true"`
	Port    int             `cfg:"port" default:"8092"`
	Path    string          `cfg:"path" default:"/metrics"`
	Timeout TimeoutSettings `cfg:"timeout"`
}

type PrometheusSettings struct {
	Aggregate bool `cfg:"aggregate" default:"false"`
	// MetricLimit is used to avoid having metrics for which the name is programmatically generated (or have large number
	// of possible dimensions) which could lead in a memory leak.
	MetricLimit int64                    `cfg:"metric_limit" default:"10000"`
	Api         PrometheusServerSettings `cfg:"api"`
}

type writerSettings struct {
	Cloudwatch CloudWatchSettings `cfg:"cloudwatch"`
	Prometheus PrometheusSettings `cfg:"prometheus"`
}

type Settings struct {
	cfg.AppId
	Enabled        bool           `cfg:"enabled" default:"false"`
	Interval       time.Duration  `cfg:"interval" default:"60s"`
	Writers        []string       `cfg:"writers"`
	WriterSettings writerSettings `cfg:"writer_settings"`
}

func getMetricSettings(config cfg.Config) *Settings {
	settings := &Settings{}
	config.UnmarshalKey("metric", settings)

	settings.PadFromConfig(config)

	return settings
}
