package ddb

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

type metricRepository struct {
	Repository
	metric mon.MetricWriter
}

func NewMetricRepository(_ cfg.Config, _ mon.Logger, repo Repository) *metricRepository {
	defaults := getDefaultMetrics(repo.GetModelId())
	output := mon.NewMetricDaemonWriter(defaults...)

	return &metricRepository{
		Repository: repo,
		metric:     output,
	}
}

func (r metricRepository) Save(ctx context.Context, item interface{}) error {
	start := time.Time{}
	err := r.Repository.Save(ctx, item)
	r.writeMetric(OpSave, err, start)

	return err
}

func (r metricRepository) writeMetric(op string, err error, start time.Time) {
	latencyNano := time.Since(start)
	modelId := r.Repository.GetModelId()
	metricName := MetricNameAccessSuccess

	if err != nil {
		metricName = MetricNameAccessFailure
	}

	r.metric.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  time.Now(),
		MetricName: metricName,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   modelId.String(),
		},
		Unit:  mon.UnitCount,
		Value: 1.0,
	})

	latencyMillisecond := float64(latencyNano) / float64(time.Millisecond)

	r.metric.WriteOne(&mon.MetricDatum{
		Timestamp:  time.Now(),
		MetricName: MetricNameAccessLatency,
		Dimensions: map[string]string{
			"Operation": op,
			"ModelId":   modelId.String(),
		},
		Unit:  mon.UnitMilliseconds,
		Value: latencyMillisecond,
	})
}

func getDefaultMetrics(mId mdl.ModelId) mon.MetricData {
	model := mId.String()
	defaults := make([]*mon.MetricDatum, 0)

	for _, op := range []string{OpSave} {
		for _, name := range []string{MetricNameAccessSuccess, MetricNameAccessFailure} {
			defaults = append(defaults, &mon.MetricDatum{
				Priority:   mon.PriorityHigh,
				MetricName: name,
				Dimensions: map[string]string{
					"Operation": op,
					"ModelId":   model,
				},
				Unit:  mon.UnitCount,
				Value: 0.0,
			})
		}

		defaults = append(defaults, &mon.MetricDatum{
			MetricName: MetricNameAccessLatency,
			Dimensions: map[string]string{
				"Operation": op,
				"ModelId":   model,
			},
			Unit:  mon.UnitMilliseconds,
			Value: 0.0,
		})
	}

	return defaults
}
