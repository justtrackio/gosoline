package ddb

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const metricNamespace = "ddb"

type metricRepository struct {
	Repository
	metric metric.Writer
}

func NewMetricRepository(config cfg.Config, logger log.Logger, repo Repository) (*metricRepository, error) {
	defaults := getDefaultMetrics(repo.GetModelId())
	output := metric.NewWriter(metricNamespace, defaults...)

	return &metricRepository{
		Repository: repo,
		metric:     output,
	}, nil
}

func (r metricRepository) PutItem(ctx context.Context, _ PutItemBuilder, item any) (*PutItemResult, error) {
	start := time.Time{}
	saved, err := r.Repository.PutItem(ctx, nil, item)
	r.writeMetric(ctx, OpSave, err, start)

	return saved, err
}

func (r metricRepository) writeMetric(ctx context.Context, op string, err error, start time.Time) {
	latencyMillisecond := float64(time.Since(start)) / float64(time.Millisecond)

	datum := &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: MetricNameOperationDuration,
		Dimensions: operationDimensions(op, r.GetModelId().String(), metric.ErrorType(err)),
		Value:      latencyMillisecond,
	}
	if err != nil {
		datum.Unit = metric.UnitMillisecondsAverage
		datum.Kind = metric.KindHistogram.Build()
	}

	r.metric.WriteOne(ctx, datum)
}

func operationDimensions(op string, modelId string, errorType string) map[string]string {
	if errorType == "" {
		errorType = metric.DimensionDefault
	}

	return map[string]string{
		dimensionOperation:        op,
		metric.DimensionModelId:   modelId,
		metric.DimensionErrorType: errorType,
	}
}

func getDefaultMetrics(mId mdl.ModelId) metric.Data {
	model := mId.String()
	defaults := make([]*metric.Datum, 0)

	for _, op := range []string{OpSave} {
		defaults = append(defaults, &metric.Datum{
			Priority:   metric.PriorityLow,
			MetricName: MetricNameOperationDuration,
			Dimensions: operationDimensions(op, model, ""),
			Unit:       metric.UnitMillisecondsAverage,
			Value:      0.0,
			Kind:       metric.KindHistogram.Build(),
		})
	}

	return defaults
}
