package metrics_per_runner

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
)

type MetricsPerRunnerEcsSettings struct {
	Cluster string `cfg:"cluster" default:"{app_project}-{env}-{app_family}"`
	Service string `cfg:"service" default:"{app_group}-{app_name}"`
}

type MetricsPerRunnerDdbServiceNamingSettings struct {
	Naming MetricsPerRunnerDdbNamingSettings `cfg:"naming"`
}

type MetricsPerRunnerDdbNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{modelId}"`
}

type MetricsPerRunnerCwServiceNamingSettings struct {
	Naming MetricsPerRunnerCwNamingSettings `cfg:"naming"`
}

type MetricsPerRunnerCwNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}/{env}/{family}/{group}-{app}"`
}

type MetricsPerRunnerMetricSettings struct {
	Ecs            MetricsPerRunnerEcsSettings              `cfg:"ecs"`
	LeaderElection string                                   `cfg:"leader_election" default:"metricsPerRunner"`
	DynamoDb       MetricsPerRunnerDdbServiceNamingSettings `cfg:"dynamodb"`
	Cloudwatch     MetricsPerRunnerCwServiceNamingSettings  `cfg:"cloudwatch"`
}

func readMetricsPerRunnerMetricSettings(config cfg.Config) *MetricsPerRunnerMetricSettings {
	mprSettings := &MetricsPerRunnerMetricSettings{}
	config.UnmarshalKey("metrics.per_runner", mprSettings)

	return mprSettings
}
