package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNamespace = "stream"

	metricNameConsumerProcessDuration = "process.duration"
	metricNameConsumedMessages        = "consumed.messages"
	metricNameConsumerRetryOperations = "retry.operations"

	dimensionConsumerName   = "consumer.name"
	dimensionRetryOperation = "operation"

	retryOperationGet = "get"
	retryOperationPut = "put"

	errorTypeProcessingFailed      = "processing_failed"
	errorTypeModelLookupFailed     = "model_lookup_failed"
	errorTypeRetryHandlerPutFailed = "retry_handler_put_failed"

	metadataKeyConsumers = "stream.consumers"
)

func init() {
	metric.RegisterHelp(metricNamespace, metricNameConsumerProcessDuration, "duration of a consumer processing operation, including batch operations")
	metric.RegisterHelp(metricNamespace, metricNameConsumedMessages, "messages a consumer processed, including overlapping error-type diagnostics")
	metric.RegisterHelp(metricNamespace, metricNameConsumerRetryOperations, "retry queue operations a consumer performed, including overlapping error diagnostics")
	metric.RegisterHelp(metricNamespace, metricNameProducedMessages, "messages a producer daemon accepted for delivery")
	metric.RegisterHelp(metricNamespace, metricNameBatchMessages, "messages a producer daemon wrote per batch")
	metric.RegisterHelp(metricNamespace, metricNameAggregateMessages, "messages a producer daemon combined into one aggregate")
	metric.RegisterHelp(metricNamespace, metricNameIdleDuration, "time a producer daemon waited before flushing a partial batch")
	metric.RegisterHelp(metricNamespace, metricNameRedisListInputMessageCount, "messages a redis list input currently holds")
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
		MetricName: metricNameConsumedMessages,
		Dimensions: map[string]string{
			dimensionConsumerName:     consumer,
			metric.DimensionErrorType: errorType,
		},
		Value: value,
		Unit:  metric.UnitCount,
		Kind:  metric.KindCounter.Build(),
	}
}

func retryOperationsDatum(consumer, operation, errorType string, value float64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameConsumerRetryOperations,
		Dimensions: map[string]string{
			dimensionConsumerName:     consumer,
			dimensionRetryOperation:   operation,
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
		retryOperationsDatum(name, retryOperationPut, metric.DimensionDefault, 0.0),
		retryOperationsDatum(name, retryOperationGet, metric.DimensionDefault, 0.0),
	}
}
