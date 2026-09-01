package kafka

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/metric"
)

// Dimension keys for all Kafka metrics. A key an OpenTelemetry semantic convention defines is spelled
// the way the convention spells it.
const (
	DimensionClientType = "kafka.client.type"
	DimensionClient     = "kafka.client.name"
	DimensionTopic      = metric.DimensionMessagingDestination
	DimensionPartition  = "messaging.destination.partition.id"
	DimensionBroker     = "kafka.broker.address"
)

// The values DimensionClientType takes, naming which side of the connection reported the metric.
const (
	ClientTypeConsumer = "consumer"
	ClientTypeProducer = "producer"
)

// MetricSpec describes one Kafka measurement, for a client of a topic partition.
type MetricSpec struct {
	ClientType string
	ClientName string
	Namespace  string
	Name       string
	Topic      string
	Partition  int32
	Value      float64
	Unit       metric.StandardUnit
	Kind       metric.Kind
}

// MetricPair writes a metric at two granularities: topic-level (KindTotal, for the CloudWatch
// cross-partition sum) and topic plus partition level. The client type and client name are always
// carried, so the Prometheus label set is the same whatever the caller reports.
func MetricPair(spec MetricSpec) metric.Data {
	topicDimensions := metric.Dimensions{
		DimensionClientType: spec.ClientType,
		DimensionClient:     spec.ClientName,
		DimensionTopic:      spec.Topic,
	}

	partitionDimensions := metric.Dimensions{
		DimensionClientType: spec.ClientType,
		DimensionClient:     spec.ClientName,
		DimensionTopic:      spec.Topic,
		DimensionPartition:  fmt.Sprintf("%d", spec.Partition),
	}

	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			Namespace:  spec.Namespace,
			MetricName: spec.Name,
			Dimensions: topicDimensions,
			Value:      spec.Value,
			Unit:       spec.Unit,
			Kind:       metric.KindTotal,
		},
		{
			Priority:   metric.PriorityHigh,
			Namespace:  spec.Namespace,
			MetricName: spec.Name,
			Dimensions: partitionDimensions,
			Value:      spec.Value,
			Unit:       spec.Unit,
			Kind:       spec.Kind,
		},
	}
}
