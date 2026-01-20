package metric

import (
	"context"
	"math"

	"github.com/justtrackio/gosoline/pkg/clock"
)

const (
	PriorityLow  = 1
	PriorityHigh = 2

	DimensionDefault = "{{default}}"
)

//go:generate go run github.com/vektra/mockery/v2 --name Writer
type (
	Writer interface {
		GetPriority() int
		Write(ctx context.Context, batch Data)
		WriteOne(ctx context.Context, data *Datum)
	}

	writer struct {
		clock   clock.Clock
		channel *metricChannel
	}
)

func NewWriter(defaults ...*Datum) Writer {
	channel := providerMetricChannel(func(*metricChannel) {})

	addMetricDefaults(defaults...)

	return NewWriterWithInterfaces(clock.Provider, channel)
}

func NewWriterWithInterfaces(clock clock.Clock, channel *metricChannel) Writer {
	return &writer{
		clock:   clock,
		channel: channel,
	}
}

func (w writer) GetPriority() int {
	return PriorityLow
}

func (w writer) Write(_ context.Context, batch Data) {
	if !w.channel.enabled || len(batch) == 0 {
		return
	}

	for i := 0; i < len(batch); i++ {
		if math.IsNaN(batch[i].Value) {
			// Replace with a counter metric for NaN occurrences
			batch[i] = &Datum{
				MetricName: "metric_writer_nan_count",
				Value:      1,
				Unit:       UnitCount,
				Dimensions: Dimensions{
					"metric_name": batch[i].MetricName,
				},
				Timestamp: w.clock.Now(),
				Priority:  PriorityHigh,
			}
		}
		if batch[i].Timestamp.IsZero() {
			batch[i].Timestamp = w.clock.Now()
		}
	}

	w.channel.write(batch)
}

func (w writer) WriteOne(ctx context.Context, data *Datum) {
	w.Write(ctx, Data{data})
}
