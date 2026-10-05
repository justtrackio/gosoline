package kafka

import (
	"fmt"

	"github.com/justtrackio/gosoline/pkg/metric"
)

// Dimension keys for all Kafka metrics. A key an OpenTelemetry semantic convention defines is spelled
// the way the convention spells it.
const (
	DimensionClientType = "client.type"
	DimensionClient     = "client.name"
	DimensionTopic      = "topic.name"
	DimensionPartition  = "partition.id"
	DimensionBroker     = "broker.address"
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
	// ErrorType, when set, is attached as the error.type attribute, so a failed operation is the same
	// metric as a successful one rather than a metric of its own. Pass metric.DimensionDefault for a
	// successful operation.
	ErrorType string
	Value     float64
	Unit      metric.StandardUnit
	Kind      metric.Kind
}

// MetricPair writes one metric for a topic partition. The client type and client name are always
// carried, so the Prometheus label set is the same whatever the caller reports.
func MetricPair(spec MetricSpec) metric.Data {
	partitionDimensions := metric.Dimensions{
		DimensionClientType: spec.ClientType,
		DimensionClient:     spec.ClientName,
		DimensionTopic:      spec.Topic,
		DimensionPartition:  fmt.Sprintf("%d", spec.Partition),
	}

	if spec.ErrorType != "" {
		partitionDimensions[metric.DimensionErrorType] = spec.ErrorType
	}

	return metric.Data{
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
