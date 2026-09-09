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

	MetricNameBrokerConnects       = "connects"
	MetricNameBrokerThrottleCount  = "throttles"
	MetricNameBrokerThrottleTime   = "throttle.duration"
	MetricNameProduceBatchRecords  = "produce.batch.records"
	MetricNameProduceBatchBytes    = "produce.batch.size"
	MetricNameProduceBatchBytesCmp = "produce.batch.compressed.size"
	MetricNameFetchBatchRecords    = "fetch.batch.records"
	MetricNameFetchBatchBytes      = "fetch.batch.size"
	MetricNameFetchBatchBytesCmp   = "fetch.batch.compressed.size"
)

func init() {
	metric.RegisterHelp(metricNamespace, MetricNameBrokerConnects, "connections a kafka client opened to a broker")
	metric.RegisterHelp(metricNamespace, MetricNameBrokerThrottleCount, "requests a broker throttled")
	metric.RegisterHelp(metricNamespace, MetricNameBrokerThrottleTime, "time a broker throttled a kafka client for")
	metric.RegisterHelp(metricNamespace, MetricNameProduceBatchRecords, "records a kafka client produced per batch")
	metric.RegisterHelp(metricNamespace, MetricNameProduceBatchBytes, "uncompressed size of a batch a kafka client produced")
	metric.RegisterHelp(metricNamespace, MetricNameProduceBatchBytesCmp, "compressed size of a batch a kafka client produced")
	metric.RegisterHelp(metricNamespace, MetricNameFetchBatchRecords, "records a kafka client fetched per batch")
	metric.RegisterHelp(metricNamespace, MetricNameFetchBatchBytes, "uncompressed size of a batch a kafka client fetched")
	metric.RegisterHelp(metricNamespace, MetricNameFetchBatchBytesCmp, "compressed size of a batch a kafka client fetched")
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

	// a failed connect is the same measurement as a successful one, told apart by its error type
	dims[metric.DimensionErrorType] = metric.ErrorType(err)
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
			MetricName: MetricNameBrokerThrottleCount,
			Dimensions: dims,
			Value:      1.0,
			Unit:       metric.UnitCount,
			Kind:       metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricNameBrokerThrottleTime,
			Dimensions: dims,
			Value:      float64(throttleInterval.Milliseconds()),
			Unit:       metric.UnitMillisecondsMaximum,
			Kind:       metric.KindHistogram.Build(),
		},
	})
}

func (h *MetricsHook) OnProduceBatchWritten(_ kgo.BrokerMetadata, topic string, partition int32, metrics kgo.ProduceBatchMetrics) {
	h.metricWriter.Write(context.Background(), metric.Data{
		h.batchDatum(MetricNameProduceBatchRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount),
		h.batchDatum(MetricNameProduceBatchBytes, topic, partition, float64(metrics.UncompressedBytes), metric.UnitBytes),
		h.batchDatum(MetricNameProduceBatchBytesCmp, topic, partition, float64(metrics.CompressedBytes), metric.UnitBytes),
	})
}

func (h *MetricsHook) OnFetchBatchRead(_ kgo.BrokerMetadata, topic string, partition int32, metrics kgo.FetchBatchMetrics) {
	h.metricWriter.Write(context.Background(), metric.Data{
		h.batchDatum(MetricNameFetchBatchRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount),
		h.batchDatum(MetricNameFetchBatchBytes, topic, partition, float64(metrics.UncompressedBytes), metric.UnitBytes),
		h.batchDatum(MetricNameFetchBatchBytesCmp, topic, partition, float64(metrics.CompressedBytes), metric.UnitBytes),
	})
}

func (h *MetricsHook) brokerDimensions(meta kgo.BrokerMetadata) metric.Dimensions {
	return metric.Dimensions{
		DimensionClientType: h.clientType,
		DimensionClient:     h.clientName,
		DimensionBroker:     fmt.Sprintf("%s:%d", meta.Host, meta.Port),
	}
}

func (h *MetricsHook) batchDatum(name string, topic string, partition int32, value float64, unit metric.StandardUnit) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: name,
		Dimensions: metric.Dimensions{
			DimensionClientType: h.clientType,
			DimensionClient:     h.clientName,
			DimensionTopic:      topic,
			DimensionPartition:  fmt.Sprintf("%d", partition),
		},
		Value: value,
		Unit:  unit,
		Kind:  metric.KindHistogram.Build(),
	}
}
