package kafka

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/metric"
)

// Dimension key constants for all Kafka metrics.
const (
	DimensionConsumer   = "Consumer"
	DimensionProducer   = "Producer"
	DimensionClientType = "ClientType"
	DimensionClient     = "Client"
	DimensionTopic      = "Topic"
	DimensionPartition  = "Partition"
	DimensionBroker     = "Broker"
)

// MetricPair writes a metric at two granularities: topic-level (KindTotal) and
// topic+partition level. Uses fixed DimensionClientType and DimensionClient labels
// to ensure consistent Prometheus label sets regardless of caller.
func MetricPair(clientType, clientName, metricName, topic string, partition int32, value float64, unit metric.StandardUnit) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricName,
			Dimensions: metric.Dimensions{DimensionClientType: clientType, DimensionClient: clientName, DimensionTopic: topic},
			Value:      value,
			Unit:       unit,
			Kind:       metric.KindTotal,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricName,
			Dimensions: metric.Dimensions{DimensionClientType: clientType, DimensionClient: clientName, DimensionTopic: topic, DimensionPartition: fmt.Sprintf("%d", partition)},
			Value:      value,
			Unit:       unit,
		},
	}
}
