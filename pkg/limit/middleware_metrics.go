package limit

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	MetricNameRateLimitRelease  = "rate_limit_release"
	MetricNameRateLimitTake     = "rate_limit_take"
	MetricNameRateLimitThrottle = "rate_limit_throttle"
	MetricNameRateLimitError    = "rate_limit_error"
)

type metricMiddleware struct {
	metricWriter metric.Writer
}

func NewMetricMiddleware() Middleware {
	metricWriter := metric.NewWriter()

	return NewMetricMiddlewareWithInterfaces(metricWriter)
}

func NewMetricMiddlewareWithInterfaces(metricWriter metric.Writer) *metricMiddleware {
	return &metricMiddleware{
		metricWriter: metricWriter,
	}
}

func (m metricMiddleware) OnTake(_ context.Context, i Invocation) {
	m.write(m.buildMetric(MetricNameRateLimitTake, i))
}

func (m metricMiddleware) OnRelease(_ context.Context, i Invocation) {
	m.write(m.buildMetric(MetricNameRateLimitRelease, i))
}

func (m metricMiddleware) OnThrottle(_ context.Context, i Invocation) {
	m.write(m.buildMetric(MetricNameRateLimitThrottle, i))
}

func (m metricMiddleware) OnError(_ context.Context, i Invocation) {
	m.write(m.buildMetric(MetricNameRateLimitError, i))
}

func (m metricMiddleware) write(metric *metric.Datum) {
	m.metricWriter.WriteOne(metric)
}

func (m metricMiddleware) buildMetric(metricName string, i Invocation) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  clock.Provider.Now(),
		MetricName: metricName,
		Dimensions: metric.Dimensions{
			"trace_id": i.GetTraceId(),
			"name":     i.GetName(),
			"prefix":   i.GetPrefix(),
		},
		Value: 1,
		Unit:  metric.UnitCount,
	}
}
