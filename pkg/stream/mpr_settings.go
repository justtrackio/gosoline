package stream

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

const (
	configKey = "stream.metrics.messages_per_runner"
)

type MessagesPerRunnerEcsSettings struct {
	Cluster string `cfg:"cluster" default:"{app_project}-{env}-{app_family}"`
	Service string `cfg:"service" default:"{app_group}-{app_name}"`
}

type MessagesPerRunnerDdbServiceNamingSettings struct {
	Naming MessagesPerRunnerDdbNamingSettings `cfg:"naming"`
}

type MessagesPerRunnerDdbNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}-{env}-{family}-{modelId}"`
}

type MessagesPerRunnerCwServiceNamingSettings struct {
	Naming MessagesPerRunnerCwNamingSettings `cfg:"naming"`
}

type MessagesPerRunnerCwNamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}/{env}/{family}/{group}-{app}"`
}

type MessagesPerRunnerMetricSettings struct {
	Enabled            bool                                      `cfg:"enabled"`
	Ecs                MessagesPerRunnerEcsSettings              `cfg:"ecs"`
	LeaderElection     string                                    `cfg:"leader_election" default:"streamMprMetrics"`
	MaxIncreasePercent float64                                   `cfg:"max_increase_percent" default:"200"`
	MaxIncreasePeriod  time.Duration                             `cfg:"max_increase_period" default:"5m"`
	DynamoDb           MessagesPerRunnerDdbServiceNamingSettings `cfg:"dynamodb"`
	Cloudwatch         MessagesPerRunnerCwServiceNamingSettings  `cfg:"cloudwatch"`
	Period             time.Duration                             `cfg:"period" default:"1m"`
	TargetValue        float64                                   `cfg:"target_value" default:"0"`
}

func readMessagesPerRunnerMetricSettings(config cfg.Config) *MessagesPerRunnerMetricSettings {
	mprSettings := &MessagesPerRunnerMetricSettings{}
	config.UnmarshalKey(configKey, mprSettings)

	return mprSettings
}
