package mdlsub

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

const (
	ConfigKeyMdlSub            = "mdlsub"
	ConfigKeyMdlSubSubscribers = "mdlsub.subscribers"
)

type Settings struct {
	Subscribers map[string]*SubscriberSettings `cfg:"subscribers"`
}

type SubscriberSettings struct {
	Input       string          `cfg:"input" default:"sns"`
	Output      string          `cfg:"output"`
	RunnerCount int             `cfg:"runner_count" default:"10" validate:"min=1"`
	SourceModel SubscriberModel `cfg:"source"`
	TargetModel SubscriberModel `cfg:"target"`
}

func unmarshalSettings(config cfg.Config) *Settings {
	settings := Settings{
		Subscribers: make(map[string]*SubscriberSettings),
	}
	config.UnmarshalKey(fmt.Sprintf("%s.%s", ConfigKeyMdlSub, "subscribers"), &settings.Subscribers)

	for name, subscriberSettings := range settings.Subscribers {
		if subscriberSettings.SourceModel.Name == "" {
			subscriberSettings.SourceModel.Name = name
		}

		if subscriberSettings.TargetModel.Name == "" {
			subscriberSettings.TargetModel.Name = name
		}
	}

	return &settings
}
