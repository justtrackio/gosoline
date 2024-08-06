package metric

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type NamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}/{env}/{family}/{group}-{app}"`
}

type Cloudwatch struct {
	Naming NamingSettings `cfg:"naming"`
}

type Settings struct {
	cfg.AppId
	Enabled    bool          `cfg:"enabled"    default:"false"`
	Interval   time.Duration `cfg:"interval"   default:"60s"`
	Cloudwatch Cloudwatch    `cfg:"cloudwatch"`
	Writer     string        `cfg:"writer"`
}

func getMetricSettings(config cfg.Config) *Settings {
	settings := &Settings{}
	config.UnmarshalKey("metric", settings)

	settings.PadFromConfig(config)

	return settings
}

const (
	promSettingsKey = "prometheus"
)

type PromSettings struct {
	// MetricLimit is used to avoid having metrics for which the name is programmatically generated (or have large number
	// of possible dimensions) which could lead in a memory leak.
	MetricLimit int64              `cfg:"metric_limit" default:"10000"`
	Api         PromServerSettings `cfg:"api"`
}

type PromServerSettings struct {
	Enabled bool            `cfg:"enabled" default:"true"`
	Port    int             `cfg:"port"    default:"8092"`
	Path    string          `cfg:"path"    default:"/metrics"`
	Timeout TimeoutSettings `cfg:"timeout"`
}

// TimeoutSettings configures IO timeouts.
type TimeoutSettings struct {
	// You need to give at least 1s as timeout.
	// Read timeout is the maximum duration for reading the entire request, including the body.
	Read time.Duration `cfg:"read"  default:"60s" validate:"min=1000000000"`
	// Write timeout is the maximum duration before timing out writes of the response.
	Write time.Duration `cfg:"write" default:"60s" validate:"min=1000000000"`
	// Idle timeout is the maximum amount of time to wait for the next request when keep-alives are enabled
	Idle time.Duration `cfg:"idle"  default:"60s" validate:"min=1000000000"`
}
