package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestMetricsHookExportsBatchVolumesAsCountersAlongsideHistograms(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheusWriter := metric.NewPrometheusWriterWithInterfaces(log.NewLogger(), registry, 1000, time.Second)
	hook := NewMetricsHook(namespaceMetricWriter{namespace: metricNamespace, delegate: prometheusWriter}, ClientTypeProducer, "producer-a")

	hook.OnProduceBatchWritten(kgo.BrokerMetadata{}, "orders", 3, kgo.ProduceBatchMetrics{
		NumRecords:        5,
		UncompressedBytes: 200,
		CompressedBytes:   100,
	})
	hook.OnProduceBatchWritten(kgo.BrokerMetadata{}, "orders", 3, kgo.ProduceBatchMetrics{
		NumRecords:        2,
		UncompressedBytes: 70,
		CompressedBytes:   30,
	})

	fetchHook := NewMetricsHook(namespaceMetricWriter{namespace: metricNamespace, delegate: prometheusWriter}, ClientTypeConsumer, "consumer-a")
	fetchHook.OnFetchBatchRead(kgo.BrokerMetadata{}, "orders", 4, kgo.FetchBatchMetrics{
		NumRecords:        3,
		UncompressedBytes: 500,
		CompressedBytes:   250,
	})
	fetchHook.OnFetchBatchRead(kgo.BrokerMetadata{}, "orders", 4, kgo.FetchBatchMetrics{
		NumRecords:        4,
		UncompressedBytes: 900,
		CompressedBytes:   450,
	})

	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 12)
	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		byName[family.GetName()] = family
	}

	producerLabels := map[string]string{
		"client_type":  "producer",
		"client_name":  "producer-a",
		"topic_name":   "orders",
		"partition_id": "3",
	}
	consumerLabels := map[string]string{
		"client_type":  "consumer",
		"client_name":  "consumer-a",
		"topic_name":   "orders",
		"partition_id": "4",
	}

	assertCounterMetric(t, byName, "gosoline_kafka_produced_records_total", 7, producerLabels)
	assertCounterMetric(t, byName, "gosoline_kafka_produced_batch_size_bytes_total", 270, producerLabels)
	assertCounterMetric(t, byName, "gosoline_kafka_produced_batch_compressed_size_bytes_total", 130, producerLabels)
	assertCounterMetric(t, byName, "gosoline_kafka_fetched_records_total", 7, consumerLabels)
	assertCounterMetric(t, byName, "gosoline_kafka_fetched_batch_size_bytes_total", 1400, consumerLabels)
	assertCounterMetric(t, byName, "gosoline_kafka_fetched_batch_compressed_size_bytes_total", 700, consumerLabels)

	assertHistogramMetric(t, byName, "gosoline_kafka_produce_batch_records", 2, 7, producerLabels)
	assertHistogramMetric(t, byName, "gosoline_kafka_produce_batch_size_bytes", 2, 270, producerLabels)
	assertHistogramMetric(t, byName, "gosoline_kafka_produce_batch_compressed_size_bytes", 2, 130, producerLabels)
	assertHistogramMetric(t, byName, "gosoline_kafka_fetch_batch_records", 2, 7, consumerLabels)
	assertHistogramMetric(t, byName, "gosoline_kafka_fetch_batch_size_bytes", 2, 1400, consumerLabels)
	assertHistogramMetric(t, byName, "gosoline_kafka_fetch_batch_compressed_size_bytes", 2, 700, consumerLabels)
}

type namespaceMetricWriter struct {
	namespace string
	delegate  metric.Writer
}

func (w namespaceMetricWriter) GetPriority() int {
	return w.delegate.GetPriority()
}

func (w namespaceMetricWriter) Write(ctx context.Context, batch metric.Data) {
	for _, datum := range batch {
		if datum.Namespace == "" {
			datum.Namespace = w.namespace
		}
	}

	w.delegate.Write(ctx, batch)
}

func (w namespaceMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
}

func assertCounterMetric(t *testing.T, families map[string]*dto.MetricFamily, name string, value float64, expectedLabels map[string]string) {
	t.Helper()
	family := requireMetricFamily(t, families, name)
	require.Equal(t, dto.MetricType_COUNTER, family.GetType())
	require.Len(t, family.Metric, 1)
	require.Equal(t, value, family.Metric[0].GetCounter().GetValue())
	require.Equal(t, expectedLabels, metricLabels(family.Metric[0]))
}

func assertHistogramMetric(t *testing.T, families map[string]*dto.MetricFamily, name string, count uint64, sum float64, expectedLabels map[string]string) {
	t.Helper()
	family := requireMetricFamily(t, families, name)
	require.Equal(t, dto.MetricType_HISTOGRAM, family.GetType())
	require.Len(t, family.Metric, 1)
	histogram := family.Metric[0].GetHistogram()
	require.Equal(t, count, histogram.GetSampleCount())
	require.Equal(t, sum, histogram.GetSampleSum())
	require.Equal(t, expectedLabels, metricLabels(family.Metric[0]))
}

func requireMetricFamily(t *testing.T, families map[string]*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()
	family, ok := families[name]
	require.Truef(t, ok, "metric family %q was not exported", name)

	return family
}

func metricLabels(sample *dto.Metric) map[string]string {
	labels := make(map[string]string, len(sample.Label))
	for _, label := range sample.Label {
		labels[label.GetName()] = label.GetValue()
	}

	return labels
}
