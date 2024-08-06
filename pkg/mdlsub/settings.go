package mdlsub

import "github.com/justtrackio/gosoline/pkg/mdl"

type PublisherSettings struct {
	mdl.ModelId
	Producer   string `cfg:"producer"    validate:"required_without=OutputType"`
	OutputType string `cfg:"output_type" validate:"required_without=Producer"`
	Shared     bool   `cfg:"shared"`
}

type SubscriberApiSettings struct {
	Enabled bool `cfg:"enabled" default:"true"`
}

type SubscriberSettings struct {
	Input       string          `cfg:"input"        default:"sns"`
	Output      string          `cfg:"output"`
	RunnerCount int             `cfg:"runner_count" default:"10"  validate:"min=1"`
	SourceModel SubscriberModel `cfg:"source"`
	TargetModel SubscriberModel `cfg:"target"`
}

type SubscriberModel struct {
	mdl.ModelId
	Shared bool `cfg:"shared"`
}

type Settings struct {
	SubscriberApi SubscriberApiSettings          `cfg:"subscriber_api"`
	Subscribers   map[string]*SubscriberSettings `cfg:"subscribers"`
}
