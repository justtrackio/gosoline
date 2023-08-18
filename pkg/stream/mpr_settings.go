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
	Service string `cfg:"service" default:"{app_name}"`
}

type MessagesPerRunnerMetricSettings struct {
	Enabled            bool                         `cfg:"enabled"`
	Ecs                MessagesPerRunnerEcsSettings `cfg:"ecs"`
	LeaderElection     string                       `cfg:"leader_election" default:"streamMprMetrics"`
	MaxIncreasePercent float64                      `cfg:"max_increase_percent" default:"200"`
	MaxIncreasePeriod  time.Duration                `cfg:"max_increase_period" default:"5m"`
	Period             time.Duration                `cfg:"period" default:"1m"`
	TargetValue        float64                      `cfg:"target_value" default:"0"`
}

func readMessagesPerRunnerMetricSettings(config cfg.Config) *MessagesPerRunnerMetricSettings {
	mprSettings := &MessagesPerRunnerMetricSettings{
		Ecs: MessagesPerRunnerEcsSettings{},
	}
	config.UnmarshalKey(configKey, mprSettings)

	return mprSettings
}

func messagesPerRunnerIsEnabled(config cfg.Config) bool {
	return config.GetBool(configKey+".enabled", false)
}
