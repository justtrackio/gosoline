package stream

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/metric/metrics_per_runner"
)

type MessagesPerRunnerMetricSettings struct {
	metrics_per_runner.HandlerSettings
	Enabled bool `cfg:"enabled" default:"false"`
}

func readMessagesPerRunnerMetricSettings(config cfg.Config) MessagesPerRunnerMetricSettings {
	mprSettings := MessagesPerRunnerMetricSettings{}
	config.UnmarshalKey("stream.metrics.messages_per_runner", &mprSettings)

	return mprSettings
}

func messagesPerRunnerIsEnabled(config cfg.Config) bool {
	return readMessagesPerRunnerMetricSettings(config).Enabled
}
