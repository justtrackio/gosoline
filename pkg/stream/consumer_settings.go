package stream

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"golang.org/x/exp/maps"
)

type ConsumerSettings struct {
	Input       string                `cfg:"input" default:"consumer" validate:"required"`
	RunnerCount int                   `cfg:"runner_count" default:"1" validate:"min=1"`
	Encoding    EncodingType          `cfg:"encoding" default:"application/json"`
	IdleTimeout time.Duration         `cfg:"idle_timeout" default:"10s"`
	Retry       ConsumerRetrySettings `cfg:"retry"`
}

type ConsumerRetrySettings struct {
	Enabled bool   `cfg:"enabled"`
	Type    string `cfg:"type" default:"sqs"`
}

func GetAllConsumerNames(config cfg.Config) []string {
	consumerMap := config.GetStringMap("stream.consumer", map[string]any{})

	return maps.Keys(consumerMap)
}

func ConfigurableConsumerKey(name string) string {
	return fmt.Sprintf("stream.consumer.%s", name)
}

func readConsumerSettings(config cfg.Config, name string) *ConsumerSettings {
	settings := &ConsumerSettings{}
	key := ConfigurableConsumerKey(name)
	config.UnmarshalKey(key, settings, cfg.UnmarshalWithDefaultForKey("encoding", defaultMessageBodyEncoding))

	return settings
}
