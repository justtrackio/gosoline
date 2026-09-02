package metric

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriterStampsNamespace(t *testing.T) {
	tests := map[string]struct {
		writerNamespace   string
		datumNamespace    string
		expectedNamespace string
	}{
		"Kinesis consumer uses its package namespace": {
			writerNamespace:   NamespaceCloudAwsKinesis,
			expectedNamespace: "cloud.aws.kinesis",
		},
		"Kinesis producer uses its package namespace": {
			writerNamespace:   NamespaceCloudAwsKinesis,
			expectedNamespace: "cloud.aws.kinesis",
		},
		"Kinesis shard reader uses its package namespace": {
			writerNamespace:   NamespaceCloudAwsKinesis,
			expectedNamespace: "cloud.aws.kinesis",
		},
		"Kafka broker uses its package namespace": {
			writerNamespace:   NamespaceKafka,
			expectedNamespace: "kafka",
		},
		"logger metrics use their package namespace": {
			writerNamespace:   NamespaceMetric,
			expectedNamespace: "metric",
		},
		"stream consumer uses its package namespace": {
			writerNamespace:   NamespaceStream,
			expectedNamespace: "stream",
		},
		"stream producer uses its package namespace": {
			writerNamespace:   NamespaceStream,
			expectedNamespace: "stream",
		},
		"stream input uses its package namespace": {
			writerNamespace:   NamespaceStream,
			expectedNamespace: "stream",
		},
		"stream Redis input uses its package namespace": {
			writerNamespace:   NamespaceStream,
			expectedNamespace: "stream",
		},
		"stream output uses its package namespace": {
			writerNamespace:   NamespaceStream,
			expectedNamespace: "stream",
		},
		"stream Redis output uses its package namespace": {
			writerNamespace:   NamespaceStream,
			expectedNamespace: "stream",
		},
		"explicit messaging namespace preserved": {
			writerNamespace:   NamespaceStream,
			datumNamespace:    NamespaceMessaging,
			expectedNamespace: NamespaceMessaging,
		},
		"semantic convention writer namespace remains exact": {
			writerNamespace:   NamespaceHttpServer,
			expectedNamespace: NamespaceHttpServer,
		},
		"absent namespace tolerated": {
			expectedNamespace: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			channel := newTestMetricChannel()
			writer := NewWriterWithInterfaces(clock.NewFakeClock(), channel, tt.writerNamespace)

			writer.WriteOne(t.Context(), &Datum{
				Priority:   PriorityHigh,
				Namespace:  tt.datumNamespace,
				MetricName: "errors",
				Unit:       UnitCount,
				Value:      1,
			})

			written := channel.read()
			require.Len(t, written, 1)
			assert.Equal(t, tt.expectedNamespace, written[0].Namespace)
		})
	}
}

func TestDeprecatedNamespaceAliasesResolveToPackageNamespaces(t *testing.T) {
	tests := map[string]struct {
		alias    string
		expected string
	}{
		"Kinesis consumer":    {NamespaceAwsKinesisConsumer, NamespaceCloudAwsKinesis},
		"Kinesis producer":    {NamespaceAwsKinesisProducer, NamespaceCloudAwsKinesis},
		"Kinesis shard":       {NamespaceAwsKinesisShard, NamespaceCloudAwsKinesis},
		"Kafka broker":        {NamespaceKafkaBroker, NamespaceKafka},
		"logger":              {NamespaceLog, NamespaceMetric},
		"stream consumer":     {NamespaceStreamConsumer, NamespaceStream},
		"stream producer":     {NamespaceStreamProducer, NamespaceStream},
		"stream input":        {NamespaceStreamInput, NamespaceStream},
		"stream Redis input":  {NamespaceStreamInputRedisList, NamespaceStream},
		"stream output":       {NamespaceStreamOutput, NamespaceStream},
		"stream Redis output": {NamespaceStreamOutputRedisList, NamespaceStream},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.alias)
		})
	}
}

func TestWriterStampsTimestamp(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	channel := newTestMetricChannel()
	writer := NewWriterWithInterfaces(clock.NewFakeClockAt(now), channel, NamespaceStream)

	writer.WriteOne(t.Context(), &Datum{
		Priority:   PriorityHigh,
		MetricName: "errors",
		Unit:       UnitCount,
		Value:      1,
	})

	written := channel.read()
	require.Len(t, written, 1)
	assert.Equal(t, now, written[0].Timestamp)
}

// TestDatumIdSeparatesNamespaces pins down that the datum id distinguishes the same leaf in two
// namespaces, so metric defaults and daemon batching never merge two different metrics.
func TestDatumIdSeparatesNamespaces(t *testing.T) {
	consumer := &Datum{Namespace: NamespaceStream, MetricName: "errors"}
	producer := &Datum{Namespace: "kafka.producer", MetricName: "errors"}

	assert.NotEqual(t, consumer.Id(), producer.Id())
}

func newTestMetricChannel() *metricChannel {
	return &metricChannel{
		hasData: make(chan struct{}, 1),
		enabled: true,
	}
}
