package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricDimensionConsumer             = "Consumer"
	metricNameConsumerDuration          = "Duration"
	metricNameConsumerError             = "Error"
	metricNameConsumerProcessedCount    = "ProcessedCount"
	metricNameConsumerRetryGetCount     = "RetryGetCount"
	metricNameConsumerRetryPutCount     = "RetryPutCount"
	metricNameConsumerUnknownModelError = "UnknownModelError"
	metadataKeyConsumers                = "stream.consumers"
)

type InitializeableCallback interface {
	Init(ctx context.Context) error
}

//go:generate go run github.com/vektra/mockery/v2 --name RunnableCallback
type RunnableCallback interface {
	Run(ctx context.Context) error
}

//go:generate go run github.com/vektra/mockery/v2 --name SchemaSettingsAwareCallback
type SchemaSettingsAwareCallback interface {
	GetSchemaSettings() (*SchemaSettings, error)
}

// IgnorableGetModelError is an interface that can be implemented by errors returned from GetModel
// to indicate whether the error is ignorable (i.e., the message should be acknowledged without processing).
type IgnorableGetModelError interface {
	error
	// IsIgnorableWithSettings returns true if this error should result in the message being ignored
	// based on the given settings.
	IsIgnorableWithSettings(settings IgnoreOnGetModelErrorSettings) bool
}

func getConsumerDefaultMetrics(name string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerProcessedCount,
			Dimensions: map[string]string{
				metricDimensionConsumer: name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerError,
			Dimensions: map[string]string{
				metricDimensionConsumer: name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryPutCount,
			Dimensions: map[string]string{
				metricDimensionConsumer: name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryGetCount,
			Dimensions: map[string]string{
				metricDimensionConsumer: name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerUnknownModelError,
			Dimensions: map[string]string{
				metricDimensionConsumer: name,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
