package stream_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/stretchr/testify/assert"
)

func TestReadConsumerSettings_Empty(t *testing.T) {
	config := cfg.New()
	settings := stream.ReadConsumerSettings(config, "defaultConsumer")
	assert.Equal(t, stream.ConsumerSettings{
		Input:       "consumer",
		RunnerCount: 1,
		Encoding:    "application/json",
		IdleTimeout: time.Second * 10,
		Retry: stream.ConsumerRetrySettings{
			Enabled:   false,
			Type:      "sqs",
			GraceTime: time.Second * 10,
		},
		Healthcheck: health.HealthCheckSettings{
			Timeout: 5 * time.Minute,
		},
	}, settings)
}

func TestReadConsumerSettings_ReadKernelKillTimeout(t *testing.T) {
	config := cfg.New(map[string]any{
		"kernel": map[string]any{
			"kill_timeout": "5s",
		},
	})
	settings := stream.ReadConsumerSettings(config, "defaultConsumer")
	assert.Equal(t, stream.ConsumerSettings{
		Input:       "consumer",
		RunnerCount: 1,
		Encoding:    "application/json",
		IdleTimeout: time.Second * 10,
		Retry: stream.ConsumerRetrySettings{
			Enabled:   false,
			Type:      "sqs",
			GraceTime: time.Second * 5,
		},
		Healthcheck: health.HealthCheckSettings{
			Timeout: 5 * time.Minute,
		},
	}, settings)
}

func TestReadConsumerSettings_SpecifyAll(t *testing.T) {
	config := cfg.New(map[string]any{
		"stream": map[string]any{
			"consumer": map[string]any{
				"defaultConsumer": map[string]any{
					"input":        "my_consumer",
					"runner_count": 2,
					"encoding":     "application/protobuf",
					"idle_timeout": "5s",
					"retry": map[string]any{
						"enabled":    true,
						"type":       "kinesis",
						"grace_time": "3s",
					},
					"healthcheck": map[string]any{
						"timeout": "3m",
					},
				},
			},
		},
		"kernel": map[string]any{
			"kill_timeout": "5s",
		},
	})
	settings := stream.ReadConsumerSettings(config, "defaultConsumer")
	assert.Equal(t, stream.ConsumerSettings{
		Input:       "my_consumer",
		RunnerCount: 2,
		Encoding:    "application/protobuf",
		IdleTimeout: time.Second * 5,
		Retry: stream.ConsumerRetrySettings{
			Enabled:   true,
			Type:      "kinesis",
			GraceTime: time.Second * 3,
		},
		Healthcheck: health.HealthCheckSettings{
			Timeout: 3 * time.Minute,
		},
	}, settings)
}
