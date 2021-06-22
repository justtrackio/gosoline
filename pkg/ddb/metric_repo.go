package ddb

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/metric"
	"time"
)

type metricRepository struct {
	Repository
	metric metric.Writer
}

func NewMetricRepository(_ cfg.Config, _ log.Logger, repo Repository) *metricRepository {
	defaults := getDefaultMetrics(repo.GetModelId())
	output := metric.NewDaemonWriter(defaults...)

	return &metricRepository{
		Repository: repo,
		metric:     output,
	}
}

func (r metricRepository) PutItem(ctx context.Context, qb PutItemBuilder, item interface{}) (*PutItemResult, error) {
	start := time.Time{}
	saved, err := r.Repository.PutItem(ctx, nil, item)
	r.writeMetric(OpSave, err, start)

	return saved, err
}

func (r metricRepository) writeMetric(op string, err error, start time.Time) {
	latencyNano := time.Since(start)
	modelId := r.Repository.GetModelId()
	metricName := MetricNameAccessSuccess

	if err != nil {
		metricName = MetricNameAccessFailure
	}

	r.metric.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   modelId.String(),
		},
		Unit:  metric.UnitCount,
		Value: 1.0,
	})

	latencyMillisecond := float64(latencyNano) / float64(time.Millisecond)

	r.metric.WriteOne(&metric.Datum{
		Timestamp:  time.Now(),
		MetricName: MetricNameAccessLatency,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   modelId.String(),
		},
		Unit:  metric.UnitMillisecondsAverage,
		Value: latencyMillisecond,
	})
}

func getDefaultMetrics(mId mdl.ModelId) metric.Data {
	model := mId.String()
	defaults := make([]*metric.Datum, 0)

	for _, op := range []string{OpSave} {
		for _, name := range []string{MetricNameAccessSuccess, MetricNameAccessFailure} {
			defaults = append(defaults, &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: name,
				Dimensions: map[string]string{
					"Operation": op,
					"ModelId":   model,
				},
				Unit:  metric.UnitCount,
				Value: 0.0,
			})
		}

		defaults = append(defaults, &metric.Datum{
			Priority:   metric.PriorityLow,
			MetricName: MetricNameAccessLatency,
			Dimensions: map[string]string{
				"Operation": op,
				"ModelId":   model,
			},
			Unit:  metric.UnitMillisecondsAverage,
			Value: 0.0,
		})
	}

	return defaults
}
