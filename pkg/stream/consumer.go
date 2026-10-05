package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNamespace = "stream"

	metricNameConsumerDuration        = "process.duration"
	metricNameConsumerProcessedCount  = "consumed.messages"
	metricNameConsumerRetryOperations = "retry.operations"

	dimensionConsumer       = "consumer.name"
	dimensionRetryOperation = "operation"

	retryOperationGet = "get"
	retryOperationPut = "put"

	errorTypeProcessingFailed = "processing_failed"

	metadataKeyConsumers = "stream.consumers"
)

func init() {
	metric.RegisterHelp(metricNamespace, metricNameConsumerDuration, "duration of a consumer processing operation, including batch operations")
	metric.RegisterHelp(metricNamespace, metricNameConsumerProcessedCount, "messages a consumer took in from its input, by error type")
	metric.RegisterHelp(metricNamespace, metricNameConsumerRetryOperations, "retry queue operations a consumer performed")
	metric.RegisterHelp(metricNamespace, metricNameMessageCount, "messages a producer daemon accepted for delivery")
	metric.RegisterHelp(metricNamespace, metricNameBatchSize, "messages a producer daemon wrote per batch")
	metric.RegisterHelp(metricNamespace, metricNameAggregateSize, "messages a producer daemon combined into one aggregate")
	metric.RegisterHelp(metricNamespace, metricNameIdleDuration, "time a producer daemon waited before flushing a partial batch")
	metric.RegisterHelp(metricNamespace, metricNameRedisListInputLength, "messages a redis list input currently holds")
	metric.RegisterHelp(metricNamespace, metricNameRedisListInputReads, "read operations a redis list input performed")
	metric.RegisterHelp(metricNamespace, metricNameRedisListOutputWrites, "write operations a redis list output performed")
}

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

func consumedMessagesDatum(consumer string, errorType string, value float64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameConsumerProcessedCount,
		Dimensions: map[string]string{
			dimensionConsumer:         consumer,
			metric.DimensionErrorType: errorType,
		},
		Value: value,
		Unit:  metric.UnitCount,
		Kind:  metric.KindCounter.Build(),
	}
}

func getConsumerDefaultMetrics(name string) metric.Data {
	return metric.Data{
		consumedMessagesDatum(name, metric.DimensionDefault, 0.0),
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryOperations,
			Dimensions: map[string]string{
				dimensionConsumer:       name,
				dimensionRetryOperation: retryOperationPut,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
			Kind:  metric.KindCounter.Build(),
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameConsumerRetryOperations,
			Dimensions: map[string]string{
				dimensionConsumer:       name,
				dimensionRetryOperation: retryOperationGet,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
			Kind:  metric.KindCounter.Build(),
		},
	}
}
