package stream

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

type ConsumerSettings struct {
	Input       string                     `cfg:"input" default:"consumer" validate:"required"`
	RunnerCount int                        `cfg:"runner_count" default:"1" validate:"min=1"`
	Encoding    EncodingType               `cfg:"encoding" default:"application/json"`
	IdleTimeout time.Duration              `cfg:"idle_timeout" default:"10s"`
	Retry       ConsumerRetrySettings      `cfg:"retry"`
	Healthcheck health.HealthCheckSettings `cfg:"healthcheck"`
}

type ConsumerRetrySettings struct {
	Enabled   bool          `cfg:"enabled"`
	Type      string        `cfg:"type" default:"sqs"`
	GraceTime time.Duration `cfg:"grace_time" default:"10s"`
}

func GetAllConsumerNames(config cfg.Config) []string {
	consumerMap := config.GetStringMap("stream.consumer", map[string]any{})

	return funk.Keys(consumerMap)
}

func ConfigurableConsumerKey(name string) string {
	return fmt.Sprintf("stream.consumer.%s", name)
}

func ReadConsumerSettings(config cfg.Config, name string) (ConsumerSettings, error) {
	settings := ConsumerSettings{}
	key := ConfigurableConsumerKey(name)
	if err := config.UnmarshalKey(
		key,
		&settings,
		cfg.UnmarshalWithDefaultForKey("encoding", defaultMessageBodyEncoding),
		// use the kernels kill timeout as the default time we allow after a cancel of the context for writing retry messages.
		// if we are processing a message and get a SIGTERM at that moment, writing the message to the retry queue will
		// fail without some time buffer for writing the message
		cfg.UnmarshalWithDefaultsFromKey("kernel.kill_timeout", "retry.grace_time"),
	); err != nil {
		return ConsumerSettings{}, fmt.Errorf("failed to unmarshal consumer settings for key %q in ReadConsumerSettings: %w", key, err)
	}

	return settings, nil
}
