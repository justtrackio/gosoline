package httpserver

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	metricsPerRunner "github.com/justtrackio/gosoline/pkg/metric/metrics_per_runner"
)

type RequestsPerRunnerMetricSettings struct {
	Enabled bool `cfg:"enabled" default:"false"`
}

func readRequestsPerRunnerMetricHandlerSettings(config cfg.Config) metricsPerRunner.HandlerSettings {
	handlerSettings := metricsPerRunner.HandlerSettings{}
	config.UnmarshalKey("metrics.per_runner.httpserver", &handlerSettings)

	return handlerSettings
}

func requestsPerRunnerIsEnabled(config cfg.Config, name string) bool {
	rprSettings := RequestsPerRunnerMetricSettings{}
	config.UnmarshalKey(fmt.Sprintf("httpserver.%s.metrics.requests_per_runner", name), &rprSettings)

	return rprSettings.Enabled
}
