package consumer_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kafka/consumer"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
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

func newTestSettings() consumer.Settings {
	return consumer.Settings{
		TopicId:     "test-topic",
		Healthcheck: health.HealthCheckSettings{Timeout: 5 * time.Minute},
	}
}

func TestNewConsumer_RegistersLifecycleManager(t *testing.T) {
	ctx := appctx.WithContainer(t.Context())
	config := newTestConfig()

	_, err := consumer.NewConsumer(ctx, config, log.NewLogger(), &noopHandler{}, newTestSettings(), "test-consumer")
	require.NoError(t, err, "NewConsumer should succeed with an application context")
}

func TestNewConsumer_FailsWithoutAppContext(t *testing.T) {
	ctx := t.Context()
	config := newTestConfig()

	_, err := consumer.NewConsumer(ctx, config, log.NewLogger(), &noopHandler{}, newTestSettings(), "test-consumer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add kafka consumer lifecycle manager")
}

type noopHandler struct{}

func (h *noopHandler) Handle(_ []*kgo.Record) {}
func (h *noopHandler) Stop()                  {}
