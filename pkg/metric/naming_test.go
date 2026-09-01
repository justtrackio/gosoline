package metric

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/stretchr/testify/assert"
)

func TestRenderCloudWatchName(t *testing.T) {
	tests := map[string]struct {
		namespace string
		leaf      string
		expected  string
	}{
		"semantic convention name": {
			namespace: "http.server",
			leaf:      "request.duration",
			expected:  "HttpServerRequestDuration",
		},
		"multi word component": {
			namespace: "aws.kinesis.shard",
			leaf:      "acquire.delay",
			expected:  "AwsKinesisShardAcquireDelay",
		},
		"underscore inside a component": {
			namespace: "stream.input.redis_list",
			leaf:      "message.count",
			expected:  "StreamInputRedisListMessageCount",
		},
		"leaf with an underscore": {
			namespace: "http.server",
			leaf:      "active_requests",
			expected:  "HttpServerActiveRequests",
		},
		"single component namespace": {
			namespace: "kvstore",
			leaf:      "reads",
			expected:  "KvstoreReads",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, renderCloudWatchName(tt.namespace, tt.leaf))
		})
	}
}

func TestRenderPrometheusName(t *testing.T) {
	tests := map[string]struct {
		namespace         string
		leaf              string
		unit              types.StandardUnit
		kind              kind
		expectedSubsystem string
		expectedName      string
	}{
		"duration": {
			namespace:         "http.server",
			leaf:              "request.duration",
			unit:              UnitMilliseconds,
			kind:              kindHistogram,
			expectedSubsystem: "http_server",
			expectedName:      "request_duration_seconds",
		},
		"counter": {
			namespace:         "stream.consumer",
			leaf:              "errors",
			unit:              UnitCount,
			kind:              kindCounter,
			expectedSubsystem: "stream_consumer",
			expectedName:      "errors_total",
		},
		"byte count": {
			namespace:         "kafka.broker",
			leaf:              "produce.batch.size",
			unit:              UnitBytes,
			kind:              kindHistogram,
			expectedSubsystem: "kafka_broker",
			expectedName:      "produce_batch_size_bytes",
		},
		"custom aggregation unit resolves to its base unit": {
			namespace:         "conc.scheduler",
			leaf:              "task.delay",
			unit:              UnitMillisecondsAverage,
			kind:              kindHistogram,
			expectedSubsystem: "conc_scheduler",
			expectedName:      "task_delay_seconds",
		},
		"unitless gauge": {
			namespace:         "kvstore",
			leaf:              "item.count",
			unit:              UnitCount,
			kind:              kindGauge,
			expectedSubsystem: "kvstore",
			expectedName:      "item_count",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			subsystem, metricName := renderPrometheusName(tt.namespace, tt.leaf, tt.unit, tt.kind)

			assert.Equal(t, tt.expectedSubsystem, subsystem)
			assert.Equal(t, tt.expectedName, metricName)
			assert.NotContains(t, subsystem, ".", "a dot is not a valid prometheus name character")
			assert.NotContains(t, metricName, ".", "a dot is not a valid prometheus name character")
		})
	}
}

func TestRenderOtelName(t *testing.T) {
	tests := map[string]struct {
		namespace string
		leaf      string
		expected  string
	}{
		"semantic convention metric": {
			namespace: "http.server",
			leaf:      "request.duration",
			expected:  "http.server.request.duration",
		},
		"gosoline specific metric": {
			namespace: "stream.consumer",
			leaf:      "errors",
			expected:  "gosoline.stream.consumer.errors",
		},
		"gosoline metric inside a semantic convention namespace": {
			namespace: "http.server",
			leaf:      "rejected.requests",
			expected:  "gosoline.http.server.rejected.requests",
		},
		"semantic convention namespace is matched exactly": {
			namespace: "db.repo",
			leaf:      "model_event.notifications",
			expected:  "gosoline.db.repo.model_event.notifications",
		},
		"semantic convention messaging metric": {
			namespace: "messaging",
			leaf:      "process.duration",
			expected:  "messaging.process.duration",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, renderOtelName(tt.namespace, tt.leaf))
		})
	}
}

// TestRenderersAreInertWithoutANamespace pins the renderers down for metrics authored outside
// gosoline: without a namespace every renderer reproduces the name it was given, so introducing the
// renderers changes no exported name before the gosoline names are authored.
func TestRenderersAreInertWithoutANamespace(t *testing.T) {
	names := []string{
		"myMetricName",
		"counter",
		"already_snake",
		"HttpRequestCount",
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, name, renderCloudWatchName("", name))

			subsystem, promName := renderPrometheusName("", name, UnitMilliseconds, kindCounter)
			assert.Empty(t, subsystem)
			assert.Equal(t, name, promName)

			assert.Equal(t, FormatOtelMetricName(name), renderOtelName("", name))
		})
	}
}
