package db_repo

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	// MetricNameDbOperationDuration records how long a repository operation took.
	MetricNameDbOperationDuration = "operation.duration"
	metricNameDbOperations        = "operations"

	// dimensionOperation is the semantic-convention attribute naming the database operation.
	dimensionOperation = "operation.name"
)

func init() {
	metric.RegisterHelp(metricNamespace, metricNameDbOperations, "SQL repository operations completed, by operation and error type")
}

type metricRepository struct {
	Repository
	output metric.Writer
}

func NewMetricRepository(_ cfg.Config, _ log.Logger, repo Repository) *metricRepository {
	modelIdString := repo.GetModelId()
	defaults := getDefaultRepositoryMetrics(modelIdString)
	output := metric.NewWriter(metricNamespace, defaults...)

	return &metricRepository{
		Repository: repo,
		output:     output,
	}
}

func (r metricRepository) Create(ctx context.Context, value ModelBased) error {
	start := time.Now()
	err := r.Repository.Create(ctx, value)
	r.writeMetric(ctx, Create, err, start)

	return err
}

func (r metricRepository) Read(ctx context.Context, id *uint, out ModelBased) error {
	start := time.Now()
	err := r.Repository.Read(ctx, id, out)
	r.writeMetric(ctx, Read, err, start)

	return err
}

func (r metricRepository) Update(ctx context.Context, value ModelBased) error {
	start := time.Now()
	err := r.Repository.Update(ctx, value)
	r.writeMetric(ctx, Update, err, start)

	return err
}

func (r metricRepository) Delete(ctx context.Context, value ModelBased) error {
	start := time.Now()
	err := r.Repository.Delete(ctx, value)
	r.writeMetric(ctx, Delete, err, start)

	return err
}

func (r metricRepository) Query(ctx context.Context, qb *QueryBuilder, result any) error {
	start := time.Now()
	err := r.Repository.Query(ctx, qb, result)
	r.writeMetric(ctx, Query, err, start)

	return err
}

func (r metricRepository) writeMetric(ctx context.Context, op string, err error, start time.Time) {
	latencyMillisecond := float64(time.Since(start)) / float64(time.Millisecond)

	errorType := ""
	if err != nil {
		errorType = errorTypeOperationFailed
	}

	modelId := r.GetModelId()
	datum := &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: MetricNameDbOperationDuration,
		Dimensions: dbOperationDimensions(op, modelId, errorType),
		Value:      latencyMillisecond,
	}
	if err != nil {
		datum.Unit = metric.UnitMillisecondsAverage
		datum.Kind = metric.KindHistogram.Build()
	}

	r.output.WriteOne(ctx, datum)

	r.output.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricNameDbOperations,
		Dimensions: dbOperationDimensions(op, modelId, errorType),
		Value:      1.0,
		Unit:       metric.UnitCount,
		Kind:       metric.KindCounter.Build(),
	})
}

func dbOperationDimensions(op string, modelId string, errorType string) map[string]string {
	if errorType == "" {
		errorType = metric.DimensionDefault
	}

	return map[string]string{
		dimensionOperation:        op,
		metric.DimensionModelId:   modelId,
		metric.DimensionErrorType: errorType,
	}
}

func getDefaultRepositoryMetrics(modelIdString string) []*metric.Datum {
	defaults := make([]*metric.Datum, 0, len(operations)*3)

	for _, op := range operations {
		defaults = append(defaults, &metric.Datum{
			Priority:   metric.PriorityLow,
			MetricName: MetricNameDbOperationDuration,
			Dimensions: dbOperationDimensions(op, modelIdString, ""),
			Unit:       metric.UnitMillisecondsAverage,
			Value:      0.0,
			Kind:       metric.KindHistogram.Build(),
		})
		defaults = append(defaults,
			&metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: metricNameDbOperations,
				Dimensions: dbOperationDimensions(op, modelIdString, ""),
				Unit:       metric.UnitCount,
				Value:      0.0,
				Kind:       metric.KindCounter.Build(),
			},
			&metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: metricNameDbOperations,
				Dimensions: dbOperationDimensions(op, modelIdString, errorTypeOperationFailed),
				Unit:       metric.UnitCount,
				Value:      0.0,
				Kind:       metric.KindCounter.Build(),
			},
		)
	}

	return defaults
}
