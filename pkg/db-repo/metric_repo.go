package db_repo

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

const (
	MetricNameDbAccessSuccess = "DbAccessSuccess"
	MetricNameDbAccessFailure = "DbAccessFailure"
	MetricNameDbAccessLatency = "DbAccessLatency"
)

type metricRepository struct {
	Repository
	output mon.MetricWriter
}

func NewMetricRepository(_ cfg.Config, _ mon.Logger, repo Repository) *metricRepository {
	defaults := getDefaultRepositoryMetrics(repo.GetMetadata().ModelId)
	output := mon.NewMetricDaemonWriter(defaults...)

	return &metricRepository{
		Repository: repo,
		output:     output,
	}
}

func (r metricRepository) Create(ctx context.Context, value ModelBased) error {
	start := time.Now()
	err := r.Repository.Create(ctx, value)
	r.writeMetric(Create, err, start)

	return err
}

func (r metricRepository) Read(ctx context.Context, id *uint, out ModelBased) error {
	start := time.Now()
	err := r.Repository.Read(ctx, id, out)
	r.writeMetric(Read, err, start)

	return err
}

func (r metricRepository) Update(ctx context.Context, value ModelBased) error {
	start := time.Now()
	err := r.Repository.Update(ctx, value)
	r.writeMetric(Update, err, start)

	return err
}

func (r metricRepository) Delete(ctx context.Context, value ModelBased) error {
	start := time.Now()
	err := r.Repository.Delete(ctx, value)
	r.writeMetric(Delete, err, start)

	return err
}

func (r metricRepository) Query(ctx context.Context, qb *QueryBuilder, result interface{}) error {
	start := time.Now()
	err := r.Repository.Query(ctx, qb, result)
	r.writeMetric(Query, err, start)

	return err
}

func (r metricRepository) writeMetric(op string, err error, start time.Time) {
	latencyNano := time.Since(start)
	metricName := MetricNameDbAccessSuccess

	if err != nil {
		metricName = MetricNameDbAccessFailure
	}

	r.output.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   r.GetModelId(),
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})

	latencyMillisecond := float64(latencyNano) / float64(time.Millisecond)

	r.output.WriteOne(&mon.MetricDatum{
		Timestamp:  time.Now(),
		MetricName: MetricNameDbAccessLatency,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   r.GetModelId(),
		},
		Unit:  mon.UnitMillisecondsAverage,
		Value: latencyMillisecond,
	})
}

func getDefaultRepositoryMetrics(modelId mdl.ModelId) []*mon.MetricDatum {
	defaults := make([]*mon.MetricDatum, 0)

	for _, op := range operations {
		for _, name := range []string{MetricNameDbAccessSuccess, MetricNameDbAccessFailure} {
			defaults = append(defaults, &mon.MetricDatum{
				Priority:   mon.PriorityHigh,
				MetricName: name,
				Dimensions: map[string]string{
					"Operation": op,
					"ModelId":   modelId.String(),
				},
				Unit:  mon.UnitCount,
				Value: 0.0,
			})
		}

		defaults = append(defaults, &mon.MetricDatum{
			Priority:   mon.PriorityLow,
			MetricName: MetricNameDbAccessLatency,
			Dimensions: map[string]string{
				"Operation": op,
				"ModelId":   modelId.String(),
			},
			Unit:  mon.UnitMillisecondsAverage,
			Value: 0.0,
		})
	}

	return defaults
}
