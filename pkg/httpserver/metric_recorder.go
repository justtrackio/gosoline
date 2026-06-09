package httpserver

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	concurrencyMetricSampleInterval = 10 * time.Second
	MetricHttpConcurrentRequests    = "HttpConcurrentRequests"
	MetricHttpOpenConnections       = "HttpOpenConnections"
)

//go:generate go run github.com/vektra/mockery/v2 --name ServerMetricRecorder
type ServerMetricRecorder interface {
	TrackRequestStarted(ctx context.Context)
	TrackRequestCompleted(ctx context.Context)
	TrackConnectionOpened(ctx context.Context)
	TrackConnectionClosed(ctx context.Context)
	Run(ctx context.Context) error
}

type serverMetricRecorder struct {
	name            string
	clock           clock.Clock
	writer          metric.Writer
	activeRequests  atomic.Int64
	openConnections atomic.Int64
	sampleInterval  time.Duration
}

func newServerMetricRecorder(name string) ServerMetricRecorder {
	defaults := getMetricRecorderDefaults(name)

	return newServerMetricRecorderWithInterfaces(name, clock.Provider, metric.NewWriter(defaults...), concurrencyMetricSampleInterval)
}

func newServerMetricRecorderWithInterfaces(name string, clock clock.Clock, writer metric.Writer, sampleInterval time.Duration) ServerMetricRecorder {
	return &serverMetricRecorder{
		name:           name,
		clock:          clock,
		writer:         writer,
		sampleInterval: sampleInterval,
	}
}

func (r *serverMetricRecorder) TrackRequestStarted(ctx context.Context) {
	r.writeConcurrentRequests(ctx, r.activeRequests.Add(1))
}

func (r *serverMetricRecorder) TrackRequestCompleted(ctx context.Context) {
	r.writeConcurrentRequests(ctx, r.activeRequests.Add(-1))
}

func (r *serverMetricRecorder) TrackConnectionOpened(ctx context.Context) {
	r.writeOpenConnections(ctx, r.openConnections.Add(1))
}

func (r *serverMetricRecorder) TrackConnectionClosed(ctx context.Context) {
	r.writeOpenConnections(ctx, r.openConnections.Add(-1))
}

func (r *serverMetricRecorder) Run(ctx context.Context) error {
	ticker := r.clock.NewTicker(r.sampleInterval)
	defer ticker.Stop()

	r.writeCurrent(ctx)

	for {
		select {
		case <-ctx.Done():
			r.writeCurrent(context.Background())

			return nil
		case <-ticker.Chan():
			r.writeCurrent(ctx)
		}
	}
}

func (r *serverMetricRecorder) writeCurrent(ctx context.Context) {
	r.writer.Write(ctx, metric.Data{
		r.buildGaugeDatum(MetricHttpConcurrentRequests, r.activeRequests.Load()),
		r.buildGaugeDatum(MetricHttpOpenConnections, r.openConnections.Load()),
	})
}

func (r *serverMetricRecorder) writeConcurrentRequests(ctx context.Context, value int64) {
	r.writer.WriteOne(ctx, r.buildGaugeDatum(MetricHttpConcurrentRequests, value))
}

func (r *serverMetricRecorder) writeOpenConnections(ctx context.Context, value int64) {
	r.writer.WriteOne(ctx, r.buildGaugeDatum(MetricHttpOpenConnections, value))
}

func (r *serverMetricRecorder) buildGaugeDatum(metricName string, value int64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricName,
		Dimensions: metric.Dimensions{
			"ServerName": r.name,
		},
		Unit:  metric.UnitCountMaximum,
		Kind:  metric.KindGauge.Build(),
		Value: float64(value),
	}
}

func getMetricRecorderDefaults(name string) metric.Data {
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpConcurrentRequests,
			Dimensions: metric.Dimensions{
				"ServerName": name,
			},
			Unit:  metric.UnitCountMaximum,
			Kind:  metric.KindGauge.Build(),
			Value: 0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: MetricHttpOpenConnections,
			Dimensions: metric.Dimensions{
				"ServerName": name,
			},
			Unit:  metric.UnitCountMaximum,
			Kind:  metric.KindGauge.Build(),
			Value: 0,
		},
	}
}
