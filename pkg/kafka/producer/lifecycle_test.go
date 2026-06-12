package producer_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/producer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig() cfg.GosoConf {
	return cfg.New(map[string]any{
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
					"brokers":     []any{"localhost:9092"},
					"tls_enabled": false,
				},
			},
		},
	})
}

func TestNewProducer_RegistersLifecycleManager(t *testing.T) {
	ctx := appctx.WithContainer(t.Context())
	config := newTestConfig()

	settings := &producer.Settings{
		TopicId:        "test-topic",
		Connection:     "default",
		MaxBatchBytes:  1048576,
		MaxBatchSize:   100,
		RequestTimeout: 10 * time.Second,
	}

	_, err := producer.NewProducer(ctx, config, log.NewLogger(), settings, "test-producer")
	require.NoError(t, err, "NewProducer should succeed with an application context")
}

func TestNewProducer_FailsWithoutAppContext(t *testing.T) {
	ctx := t.Context()
	config := newTestConfig()

	settings := &producer.Settings{
		TopicId:        "test-topic",
		Connection:     "default",
		MaxBatchBytes:  1048576,
		MaxBatchSize:   100,
		RequestTimeout: 10 * time.Second,
	}

	_, err := producer.NewProducer(ctx, config, log.NewLogger(), settings, "test-producer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add kafka producer lifecycle manager")
}
