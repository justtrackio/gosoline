package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	metricNamespace = "kafka"

	MetricNameBrokerConnects             = "connects"
	MetricNameBrokerThrottles            = "throttles"
	MetricNameBrokerThrottleDuration     = "throttle.duration"
	MetricNameProduceBatchRecords        = "produce.batch.records"
	MetricNameProduceBatchSize           = "produce.batch.size"
	MetricNameProduceBatchCompressedSize = "produce.batch.compressed.size"
	MetricNameFetchBatchRecords          = "fetch.batch.records"
	MetricNameFetchBatchSize             = "fetch.batch.size"
	MetricNameFetchBatchCompressedSize   = "fetch.batch.compressed.size"

	metricNameProducedRecords             = "produced.records"
	metricNameProducedBatchSize           = "produced.batch.size"
	metricNameProducedBatchCompressedSize = "produced.batch.compressed.size"
	metricNameFetchedRecords              = "fetched.records"
	metricNameFetchedBatchSize            = "fetched.batch.size"
	metricNameFetchedBatchCompressedSize  = "fetched.batch.compressed.size"

	errorTypeConnectFailed = "connect_failed"
)

func init() {
	metric.RegisterHelp(metricNamespace, MetricNameBrokerConnects, "connections a kafka client opened to a broker")
	metric.RegisterHelp(metricNamespace, MetricNameBrokerThrottles, "requests a broker throttled")
	metric.RegisterHelp(metricNamespace, MetricNameBrokerThrottleDuration, "time a broker throttled a kafka client for")
	metric.RegisterHelp(metricNamespace, MetricNameProduceBatchRecords, "records a kafka client produced per batch")
	metric.RegisterHelp(metricNamespace, MetricNameProduceBatchSize, "uncompressed size of a batch a kafka client produced")
	metric.RegisterHelp(metricNamespace, MetricNameProduceBatchCompressedSize, "compressed size of a batch a kafka client produced")
	metric.RegisterHelp(metricNamespace, MetricNameFetchBatchRecords, "records a kafka client fetched per batch")
	metric.RegisterHelp(metricNamespace, MetricNameFetchBatchSize, "uncompressed size of a batch a kafka client fetched")
	metric.RegisterHelp(metricNamespace, MetricNameFetchBatchCompressedSize, "compressed size of a batch a kafka client fetched")
	metric.RegisterHelp(metricNamespace, metricNameProducedRecords, "records a kafka client produced")
	metric.RegisterHelp(metricNamespace, metricNameProducedBatchSize, "uncompressed bytes a kafka client produced")
	metric.RegisterHelp(metricNamespace, metricNameProducedBatchCompressedSize, "compressed bytes a kafka client produced")
	metric.RegisterHelp(metricNamespace, metricNameFetchedRecords, "records a kafka client fetched")
	metric.RegisterHelp(metricNamespace, metricNameFetchedBatchSize, "uncompressed bytes a kafka client fetched")
	metric.RegisterHelp(metricNamespace, metricNameFetchedBatchCompressedSize, "compressed bytes a kafka client fetched")
}

// MetricsHook implements franz-go hook interfaces to emit metrics for broker
// connectivity, throttling, and produce/consume batch operations.
type MetricsHook struct {
	metricWriter metric.Writer
	clientType   string
	clientName   string
}

// Compile-time interface assertions.
var (
	_ kgo.HookBrokerConnect       = (*MetricsHook)(nil)
	_ kgo.HookBrokerThrottle      = (*MetricsHook)(nil)
	_ kgo.HookProduceBatchWritten = (*MetricsHook)(nil)
	_ kgo.HookFetchBatchRead      = (*MetricsHook)(nil)
)

func NewMetricsHook(metricWriter metric.Writer, clientType, clientName string) *MetricsHook {
	return &MetricsHook{metricWriter: metricWriter, clientType: clientType, clientName: clientName}
}

func (h *MetricsHook) OnBrokerConnect(meta kgo.BrokerMetadata, _ time.Duration, _ net.Conn, err error) {
	dims := h.brokerDimensions(meta)

	// a failed connect is the same measurement as a successful one, marked by the action that failed
	dims[metric.DimensionErrorType] = errorTypeConnectFailed
	if err == nil {
		dims[metric.DimensionErrorType] = metric.DimensionDefault
	}

	h.metricWriter.WriteOne(context.Background(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: MetricNameBrokerConnects,
		Dimensions: dims,
		Value:      1.0,
		Unit:       metric.UnitCount,
		Kind:       metric.KindCounter.Build(),
	})
}

func (h *MetricsHook) OnBrokerThrottle(meta kgo.BrokerMetadata, throttleInterval time.Duration, _ bool) {
	dims := h.brokerDimensions(meta)

	h.metricWriter.Write(context.Background(), metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricNameBrokerThrottles,
			Dimensions: dims,
			Value:      1.0,
			Unit:       metric.UnitCount,
			Kind:       metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricNameBrokerThrottleDuration,
			Dimensions: dims,
			Value:      float64(throttleInterval.Milliseconds()),
			Unit:       metric.UnitMillisecondsMaximum,
			Kind:       metric.KindHistogram.Build(),
		},
	})
}

func (h *MetricsHook) OnProduceBatchWritten(_ kgo.BrokerMetadata, topic string, partition int32, metrics kgo.ProduceBatchMetrics) {
	h.metricWriter.Write(context.Background(), metric.Data{
		h.batchDatum(MetricNameProduceBatchRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount, metric.KindHistogram.Build()),
		h.batchDatum(MetricNameProduceBatchSize, topic, partition, float64(metrics.UncompressedBytes), metric.UnitBytes, metric.KindHistogram.Build()),
		h.batchDatum(MetricNameProduceBatchCompressedSize, topic, partition, float64(metrics.CompressedBytes), metric.UnitBytes, metric.KindHistogram.Build()),
		h.batchDatum(metricNameProducedRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount, metric.KindCounter.Build()),
		h.batchDatum(metricNameProducedBatchSize, topic, partition, float64(metrics.UncompressedBytes), metric.UnitBytes, metric.KindCounter.Build()),
		h.batchDatum(metricNameProducedBatchCompressedSize, topic, partition, float64(metrics.CompressedBytes), metric.UnitBytes, metric.KindCounter.Build()),
	})
}

func (h *MetricsHook) OnFetchBatchRead(_ kgo.BrokerMetadata, topic string, partition int32, metrics kgo.FetchBatchMetrics) {
	h.metricWriter.Write(context.Background(), metric.Data{
		h.batchDatum(MetricNameFetchBatchRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount, metric.KindHistogram.Build()),
		h.batchDatum(MetricNameFetchBatchSize, topic, partition, float64(metrics.UncompressedBytes), metric.UnitBytes, metric.KindHistogram.Build()),
		h.batchDatum(MetricNameFetchBatchCompressedSize, topic, partition, float64(metrics.CompressedBytes), metric.UnitBytes, metric.KindHistogram.Build()),
		h.batchDatum(metricNameFetchedRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount, metric.KindCounter.Build()),
		h.batchDatum(metricNameFetchedBatchSize, topic, partition, float64(metrics.UncompressedBytes), metric.UnitBytes, metric.KindCounter.Build()),
		h.batchDatum(metricNameFetchedBatchCompressedSize, topic, partition, float64(metrics.CompressedBytes), metric.UnitBytes, metric.KindCounter.Build()),
	})
}

func (h *MetricsHook) brokerDimensions(meta kgo.BrokerMetadata) metric.Dimensions {
	return metric.Dimensions{
		DimensionClientType:    h.clientType,
		DimensionClientName:    h.clientName,
		DimensionBrokerAddress: fmt.Sprintf("%s:%d", meta.Host, meta.Port),
	}
}

func (h *MetricsHook) batchDatum(name string, topic string, partition int32, value float64, unit metric.StandardUnit, kind metric.Kind) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: name,
		Dimensions: metric.Dimensions{
			DimensionClientType:  h.clientType,
			DimensionClientName:  h.clientName,
			DimensionTopicName:   topic,
			DimensionPartitionId: fmt.Sprintf("%d", partition),
		},
		Value: value,
		Unit:  unit,
		Kind:  kind,
	}
}
