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
		"name a semantic convention also defines is still prefixed": {
			namespace: "http.server",
			leaf:      "request.duration",
			expected:  "gosoline.http.server.request.duration",
		},
		"gosoline specific metric": {
			namespace: "stream.consumer",
			leaf:      "errors",
			expected:  "gosoline.stream.consumer.errors",
		},
		"multi component leaf": {
			namespace: "http.server",
			leaf:      "rejected.requests",
			expected:  "gosoline.http.server.rejected.requests",
		},
		"compound word inside a leaf": {
			namespace: "db.repo",
			leaf:      "model_event.notifications",
			expected:  "gosoline.db.repo.model_event.notifications",
		},
		"per transport processing metric": {
			namespace: "kafka.consumer",
			leaf:      "process.duration",
			expected:  "gosoline.kafka.consumer.process.duration",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, renderOtelName(tt.namespace, tt.leaf))
		})
	}
}

// TestRenderersConvertALeafWithoutANamespace pins the renderers down for metrics authored outside
// gosoline: a datum without a namespace still has its leaf converted into the convention each writer
// exports under, because a leaf carries canonical separators whether or not a namespace precedes it.
func TestRenderersConvertALeafWithoutANamespace(t *testing.T) {
	tests := map[string]struct {
		leaf               string
		expectedCloudWatch string
		expectedPrometheus string
	}{
		"single word": {
			leaf:               "counter",
			expectedCloudWatch: "Counter",
			expectedPrometheus: "counter_seconds_total",
		},
		"camel case": {
			leaf:               "myMetricName",
			expectedCloudWatch: "MyMetricName",
			expectedPrometheus: "myMetricName_seconds_total",
		},
		"underscore separated": {
			leaf:               "already_snake",
			expectedCloudWatch: "AlreadySnake",
			expectedPrometheus: "already_snake_seconds_total",
		},
		"dotted leaf": {
			leaf:               "request.duration",
			expectedCloudWatch: "RequestDuration",
			expectedPrometheus: "request_duration_seconds_total",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expectedCloudWatch, renderCloudWatchName("", tt.leaf))

			subsystem, promName := renderPrometheusName("", tt.leaf, UnitMilliseconds, kindCounter)
			assert.Empty(t, subsystem, "a datum without a namespace has no prometheus subsystem")
			assert.Equal(t, tt.expectedPrometheus, promName)
			assert.NotContains(t, promName, ".", "a dot is not a valid prometheus name character")

			assert.Equal(t, FormatOtelMetricName(tt.leaf), renderOtelName("", tt.leaf))
		})
	}
}
