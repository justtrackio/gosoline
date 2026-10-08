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
			writerNamespace:   "cloud.aws.kinesis",
			expectedNamespace: "cloud.aws.kinesis",
		},
		"Kinesis producer uses its package namespace": {
			writerNamespace:   "cloud.aws.kinesis",
			expectedNamespace: "cloud.aws.kinesis",
		},
		"Kinesis shard reader uses its package namespace": {
			writerNamespace:   "cloud.aws.kinesis",
			expectedNamespace: "cloud.aws.kinesis",
		},
		"Kafka broker uses its package namespace": {
			writerNamespace:   "kafka",
			expectedNamespace: "kafka",
		},
		"logger metrics use their package namespace": {
			writerNamespace:   "metric",
			expectedNamespace: "metric",
		},
		"stream consumer uses its package namespace": {
			writerNamespace:   "stream",
			expectedNamespace: "stream",
		},
		"stream producer uses its package namespace": {
			writerNamespace:   "stream",
			expectedNamespace: "stream",
		},
		"stream input uses its package namespace": {
			writerNamespace:   "stream",
			expectedNamespace: "stream",
		},
		"stream Redis input uses its package namespace": {
			writerNamespace:   "stream",
			expectedNamespace: "stream",
		},
		"stream output uses its package namespace": {
			writerNamespace:   "stream",
			expectedNamespace: "stream",
		},
		"stream Redis output uses its package namespace": {
			writerNamespace:   "stream",
			expectedNamespace: "stream",
		},
		"explicit messaging namespace preserved": {
			writerNamespace:   "stream",
			datumNamespace:    "messaging",
			expectedNamespace: "messaging",
		},
		"semantic convention writer namespace remains exact": {
			writerNamespace:   "http.server",
			expectedNamespace: "http.server",
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

func TestWriterStampsTimestamp(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	channel := newTestMetricChannel()
	writer := NewWriterWithInterfaces(clock.NewFakeClockAt(now), channel, "stream")

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
	consumer := &Datum{Namespace: "stream", MetricName: "errors"}
	producer := &Datum{Namespace: "kafka.producer", MetricName: "errors"}

	assert.NotEqual(t, consumer.Id(), producer.Id())
}

func newTestMetricChannel() *metricChannel {
	return &metricChannel{
		hasData: make(chan struct{}, 1),
		enabled: true,
	}
}
