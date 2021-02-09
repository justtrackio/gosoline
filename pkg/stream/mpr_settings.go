package stream

import (
	"github.com/applike/gosoline/pkg/cfg"
	"time"
)

type MessagesPerRunnerMetricSettings struct {
	Enabled            bool          `cfg:"enabled"`
	LeaderElection     string        `cfg:"leader_election" default:"streamMprMetrics"`
	Period             time.Duration `cfg:"period" default:"1m"`
	MaxIncreasePercent float64       `cfg:"max_increase_percent" default:"200"`
	MaxIncreasePeriod  time.Duration `cfg:"max_increase_period" default:"5m"`
}

func readMessagesPerRunnerMetricSettings(config cfg.Config) *MessagesPerRunnerMetricSettings {
	mprSettings := &MessagesPerRunnerMetricSettings{}
	config.UnmarshalKey("stream.metrics.messages_per_runner", mprSettings)

	return mprSettings
}
