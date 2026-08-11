package stream

import (
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/stream/health"
)

const (
	AggregateMessageModeAtLeastOnce = "atLeastOnce"
	AggregateMessageModeAtMostOnce  = "atMostOnce"
)

type ConsumerSettings struct {
	Input       string        `cfg:"input" default:"consumer" validate:"required"`
	Encoding    EncodingType  `cfg:"encoding" default:"application/json"`
	IdleTimeout time.Duration `cfg:"idle_timeout" default:"10s"`
	// GraceTime is the maximum time a record has to be processed once the consumer stops. It is the single
	// authoritative processing deadline and applies to every input, both the primary and the retry one: inputs must
	// stop fetching new messages, while messages they already fetched are processed until this shared deadline. When
	// it expires, in-flight callback contexts are canceled so the inputs can nack or retry any remaining messages.
	//
	// Inputs do not define a processing deadline of their own. Their own grace_time bounds how long they get to
	// acknowledge or commit what was processed, which is a window that only starts once this deadline expired.
	GraceTime             time.Duration                 `cfg:"grace_time" default:"10s"`
	Retry                 ConsumerRetrySettings         `cfg:"retry"`
	Healthcheck           health.HealthCheckSettings    `cfg:"healthcheck"`
	AggregateMessageMode  string                        `cfg:"aggregate_message_mode" default:"atMostOnce" validate:"oneof=atLeastOnce atMostOnce"`
	IgnoreOnGetModelError IgnoreOnGetModelErrorSettings `cfg:"ignore_on_get_model_error"`
}

// IgnoreOnGetModelErrorSettings configures which GetModel errors should result in the message being ignored
// (acknowledged without processing) rather than being treated as an error.
type IgnoreOnGetModelErrorSettings struct {
	// UnknownModel indicates whether to ignore messages when the model is unknown.
	// When true, messages with unknown model IDs will be acknowledged and skipped.
	UnknownModel bool `cfg:"unknown_model" default:"false"`
	// UnknownVersion indicates whether to ignore messages when the version is unknown.
	// When true, messages with unknown versions for known models will be acknowledged and skipped.
	UnknownVersion bool `cfg:"unknown_version" default:"false"`
}

type ConsumerRetrySettings struct {
	Enabled   bool          `cfg:"enabled"`
	Type      string        `cfg:"type" default:"sqs"`
	GraceTime time.Duration `cfg:"grace_time" default:"10s"`
}

func GetAllConsumerNames(config cfg.Config) ([]string, error) {
	consumerMap, err := config.GetStringMap("stream.consumer", map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer settings: %w", err)
	}

	return funk.Keys(consumerMap), nil
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
