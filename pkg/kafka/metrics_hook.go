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
	MetricNameBrokerConnects       = "BrokerConnects"
	MetricNameBrokerConnectsFailed = "BrokerConnectsFailed"
	MetricNameBrokerThrottleCount  = "BrokerThrottleCount"
	MetricNameBrokerThrottleTime   = "BrokerThrottleTime"
	MetricNameProduceBatchRecords  = "ProduceBatchRecords"
	MetricNameProduceBatchBytes    = "ProduceBatchBytes"
	MetricNameProduceBatchBytesCmp = "ProduceBatchBytesCompressed"
	MetricNameFetchBatchRecords    = "FetchBatchRecords"
	MetricNameFetchBatchBytes      = "FetchBatchBytes"
	MetricNameFetchBatchBytesCmp   = "FetchBatchBytesCompressed"
)

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
	dims := metric.Dimensions{
		DimensionClientType: h.clientType,
		DimensionClient:     h.clientName,
		DimensionBroker:     fmt.Sprintf("%s:%d", meta.Host, meta.Port),
	}

	metricName := MetricNameBrokerConnects
	if err != nil {
		metricName = MetricNameBrokerConnectsFailed
	}

	h.metricWriter.WriteOne(context.Background(), metric.NewMetricDatum(metricName, dims, 1.0, metric.UnitCount, metric.PriorityHigh))
}

func (h *MetricsHook) OnBrokerThrottle(meta kgo.BrokerMetadata, throttleInterval time.Duration, _ bool) {
	dims := metric.Dimensions{
		DimensionClientType: h.clientType,
		DimensionClient:     h.clientName,
		DimensionBroker:     fmt.Sprintf("%s:%d", meta.Host, meta.Port),
	}

	h.metricWriter.Write(context.Background(), metric.Data{
		metric.NewMetricDatum(MetricNameBrokerThrottleCount, dims, 1.0, metric.UnitCount, metric.PriorityHigh),
		metric.NewMetricDatum(MetricNameBrokerThrottleTime, dims, float64(throttleInterval.Milliseconds()), metric.UnitMillisecondsMaximum, metric.PriorityHigh),
	})
}

func (h *MetricsHook) OnProduceBatchWritten(_ kgo.BrokerMetadata, topic string, partition int32, metrics kgo.ProduceBatchMetrics) {
	var data metric.Data
	data = append(data, MetricPair(h.clientType, h.clientName, MetricNameProduceBatchRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount)...)
	data = append(data, MetricPair(h.clientType, h.clientName, MetricNameProduceBatchBytes, topic, partition, float64(metrics.UncompressedBytes), metric.UnitCount)...)
	data = append(data, MetricPair(h.clientType, h.clientName, MetricNameProduceBatchBytesCmp, topic, partition, float64(metrics.CompressedBytes), metric.UnitCount)...)

	h.metricWriter.Write(context.Background(), data)
}

func (h *MetricsHook) OnFetchBatchRead(_ kgo.BrokerMetadata, topic string, partition int32, metrics kgo.FetchBatchMetrics) {
	var data metric.Data
	data = append(data, MetricPair(h.clientType, h.clientName, MetricNameFetchBatchRecords, topic, partition, float64(metrics.NumRecords), metric.UnitCount)...)
	data = append(data, MetricPair(h.clientType, h.clientName, MetricNameFetchBatchBytes, topic, partition, float64(metrics.UncompressedBytes), metric.UnitCount)...)
	data = append(data, MetricPair(h.clientType, h.clientName, MetricNameFetchBatchBytesCmp, topic, partition, float64(metrics.CompressedBytes), metric.UnitCount)...)

	h.metricWriter.Write(context.Background(), data)
}
