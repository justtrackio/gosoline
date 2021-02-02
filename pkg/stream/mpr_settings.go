package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"time"
)

type MessagesPerRunnerMetricSettings struct {
	Consumers []string      `cfg:"consumers"`
	Period    time.Duration `cfg:"period" default:"1m"`
}

func readAllMessagesPerRunnerMetricSettings(config cfg.Config) map[string]*MessagesPerRunnerMetricSettings {
	mprSettings := make(map[string]*MessagesPerRunnerMetricSettings)
	producerMap := config.GetStringMap("stream.metrics.messages_per_runner", map[string]interface{}{})

	for name := range producerMap {
		mprSettings[name] = readMessagesPerRunnerMetricSettings(config, name)
	}

	return mprSettings
}

func readMessagesPerRunnerMetricSettings(config cfg.Config, name string) *MessagesPerRunnerMetricSettings {
	key := fmt.Sprintf("stream.metrics.messages_per_runner.%s", name)
	mprSettings := &MessagesPerRunnerMetricSettings{}
	config.UnmarshalKey(key, mprSettings)

	return mprSettings
}
