package db_repo

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	MetricNameDbAccessSuccess = "DbAccessSuccess"
	MetricNameDbAccessFailure = "DbAccessFailure"
	MetricNameDbAccessLatency = "DbAccessLatency"
)

type metricRepository[K mdl.PossibleIdentifier, M ModelBased[K]] struct {
	Repository[K, M]
	output metric.Writer
}

func NewMetricRepository[K mdl.PossibleIdentifier, M ModelBased[K]](_ cfg.Config, _ log.Logger, repo Repository[K, M]) Repository[K, M] {
	defaults := getDefaultRepositoryMetrics(repo.GetMetadata().ModelId)
	output := metric.NewWriter(defaults...)

	return &metricRepository[K, M]{
		Repository: repo,
		output:     output,
	}
}

func (r metricRepository[K, M]) Create(ctx context.Context, value M) error {
	start := time.Now()
	err := r.Repository.Create(ctx, value)
	r.writeMetric(ctx, Create, err, start)

	return err
}

func (r metricRepository[K, M]) Read(ctx context.Context, id K) (M, error) {
	start := time.Now()
	result, err := r.Repository.Read(ctx, id)
	r.writeMetric(ctx, Read, err, start)

	return result, err
}

func (r metricRepository[K, M]) Update(ctx context.Context, value M) error {
	start := time.Now()
	err := r.Repository.Update(ctx, value)
	r.writeMetric(ctx, Update, err, start)

	return err
}

func (r metricRepository[K, M]) Delete(ctx context.Context, value M) error {
	start := time.Now()
	err := r.Repository.Delete(ctx, value)
	r.writeMetric(ctx, Delete, err, start)

	return err
}

func (r metricRepository[K, M]) Query(ctx context.Context, qb *QueryBuilder) ([]M, error) {
	start := time.Now()
	result, err := r.Repository.Query(ctx, qb)
	r.writeMetric(ctx, Query, err, start)

	return result, err
}

func (r metricRepository[K, M]) writeMetric(ctx context.Context, op string, err error, start time.Time) {
	latencyNano := time.Since(start)
	metricName := MetricNameDbAccessSuccess

	if err != nil {
		metricName = MetricNameDbAccessFailure
	}

	r.output.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   r.GetModelId(),
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})

	latencyMillisecond := float64(latencyNano) / float64(time.Millisecond)

	r.output.WriteOne(ctx, &metric.Datum{
		Timestamp:  time.Now(),
		MetricName: MetricNameDbAccessLatency,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   r.GetModelId(),
		},
		Unit:  metric.UnitMillisecondsAverage,
		Value: latencyMillisecond,
	})
}

func getDefaultRepositoryMetrics(modelId mdl.ModelId) []*metric.Datum {
	defaults := make([]*metric.Datum, 0)

	for _, op := range operations {
		for _, name := range []string{MetricNameDbAccessSuccess, MetricNameDbAccessFailure} {
			defaults = append(defaults, &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: name,
				Dimensions: map[string]string{
					"Operation": op,
					"ModelId":   modelId.String(),
				},
				Unit:  metric.UnitCount,
				Value: 0.0,
			})
		}

		defaults = append(defaults, &metric.Datum{
			Priority:   metric.PriorityLow,
			MetricName: MetricNameDbAccessLatency,
			Dimensions: map[string]string{
				"Operation": op,
				"ModelId":   modelId.String(),
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: 0.0,
		})
	}

	return defaults
}
