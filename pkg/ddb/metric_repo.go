package ddb

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNamespace      = "ddb"
	metricNameOperations = "operations"
)

const errorTypeOperationFailed = "operation_failed"

func init() {
	metric.RegisterHelp(metricNamespace, MetricNameOperationDuration, "duration of a DynamoDB repository operation")
	metric.RegisterHelp(metricNamespace, metricNameOperations, "DynamoDB repository operations completed, by operation and error type")
}

type metricRepository struct {
	Repository
	metric metric.Writer
}

func NewMetricRepository(config cfg.Config, logger log.Logger, repo Repository) (*metricRepository, error) {
	output := metric.NewWriter(metricNamespace)

	return &metricRepository{
		Repository: repo,
		metric:     output,
	}, nil
}

func (r metricRepository) PutItem(ctx context.Context, _ PutItemBuilder, item any) (*PutItemResult, error) {
	start := time.Now()
	saved, err := r.Repository.PutItem(ctx, nil, item)
	r.writeMetric(ctx, OpSave, err, start)

	return saved, err
}

func (r metricRepository) writeMetric(ctx context.Context, op string, err error, start time.Time) {
	latencyMillisecond := float64(time.Since(start)) / float64(time.Millisecond)

	errorType := ""
	if err != nil {
		errorType = errorTypeOperationFailed
	}

	modelId := r.GetModelId().String()
	datum := &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: MetricNameOperationDuration,
		Dimensions: operationDimensions(op, modelId, errorType),
		Value:      latencyMillisecond,
		Unit:       metric.UnitMillisecondsAverage,
		Kind:       metric.KindHistogram.WithDatabaseDurationBuckets().Build(),
	}

	r.metric.WriteOne(ctx, datum)

	r.metric.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricNameOperations,
		Dimensions: operationDimensions(op, modelId, errorType),
		Value:      1.0,
		Unit:       metric.UnitCount,
		Kind:       metric.KindCounter.Build(),
	})
}

func operationDimensions(op string, modelId string, errorType string) map[string]string {
	if errorType == "" {
		errorType = metric.DimensionDefault
	}

	return map[string]string{
		dimensionOperationName:    op,
		metric.DimensionModelId:   modelId,
		metric.DimensionErrorType: errorType,
	}
}
