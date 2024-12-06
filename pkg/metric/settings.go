package metric

import (
	"fmt"
	"strings"
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
	Enabled    bool          `cfg:"enabled" default:"false"`
	Interval   time.Duration `cfg:"interval" default:"60s"`
	Cloudwatch Cloudwatch    `cfg:"cloudwatch"`
	Writer     []string      `cfg:"writer"`
}

func getMetricSettings(config cfg.Config) *Settings {
	settings := &Settings{}
	config.UnmarshalKey("metric", settings)

	settings.PadFromConfig(config)

	return settings
}

func GetCloudWatchNamespace(config cfg.Config) string {
	settings := getMetricSettings(config)
	namespace := settings.Cloudwatch.Naming.Pattern
	appId := settings.AppId

	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		namespace = strings.ReplaceAll(namespace, templ, val)
	}

	return namespace
}
