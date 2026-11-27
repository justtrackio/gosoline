package stream_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	kafkaConsumer "github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKafkaInputShareGroupOwnsRetries guards against a failed share group record being retried twice: once by Kafka,
// which redelivers every released record, and once more through the consumer's retry queue.
func TestKafkaInputShareGroupOwnsRetries(t *testing.T) {
	for groupType, ownsRetries := range map[kafkaConsumer.GroupType]bool{
		kafkaConsumer.GroupTypeClassic: false,
		kafkaConsumer.GroupTypeShare:   true,
	} {
		t.Run(string(groupType), func(t *testing.T) {
			config := cfg.New(map[string]any{
				"app": map[string]any{"env": "test", "name": "testapp"},
				"kafka": map[string]any{
					"naming": map[string]any{
						"topic_pattern": "{app.name}-{topicId}",
						"group_pattern": "{app.name}-{groupId}",
					},
					"connection": map[string]any{
						"default": map[string]any{"brokers": []any{"localhost:9092"}},
					},
				},
			})

			settings := &kafkaConsumer.Settings{
				Connection:     "default",
				TopicId:        "events",
				GroupType:      groupType,
				MaxPollRecords: 100,
				IdleWaitTime:   time.Second,
				GraceTime:      time.Second,
				Healthcheck:    health.HealthCheckSettings{Timeout: time.Minute},
			}

			input, err := stream.NewKafkaInput(appctx.WithContainer(t.Context()), config, log.NewLogger(), settings, "test")
			require.NoError(t, err)

			retryingInput, ok := input.(stream.RetryingInput)
			require.Equal(t, ownsRetries, ok)

			if ok {
				_, retryHandler := retryingInput.GetRetryHandler()
				assert.IsType(t, stream.RetryHandlerNoop{}, retryHandler)
			}
		})
	}
}
