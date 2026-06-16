package consumer_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestNewReader_KeepsRetryableFetchErrors(t *testing.T) {
	config := cfg.New(map[string]any{
		"app": map[string]any{
			"env":  "test",
			"name": "testapp",
		},
		"kafka": map[string]any{
			"naming": map[string]any{
				"topic_pattern": "{app.name}-{topicId}",
			},
			"connection": map[string]any{
				"default": map[string]any{
					"brokers": []any{"localhost:9092"},
				},
			},
		},
	})

	settings := consumer.Settings{
		Connection:       "default",
		TopicId:          "test-topic",
		SessionTimeout:   45 * time.Second,
		RebalanceTimeout: 60 * time.Second,
	}

	// isReadOnly is true so the reader does not require a consumer group / partition manager.
	reader, err := consumer.NewReader(t.Context(), config, log.NewLogger(), settings, nil, true, "test-consumer")
	require.NoError(t, err)

	client, ok := reader.(*kgo.Client)
	require.True(t, ok, "expected reader to be a *kgo.Client")

	assert.Equal(t, true, client.OptValue(kgo.KeepRetryableFetchErrors))
}
