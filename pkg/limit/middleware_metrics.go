package limit

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNamespace = "limit"

	MetricNameRateLimitTake           = "takes"
	metricNameRateLimitAttemptedTake  = "attempted.takes"
	metricNameRateLimitThrottledTakes = "throttled.takes"

	// dimensionOutcome distinguishes the mutually exclusive terminal result of a take.
	dimensionOutcome = "outcome"

	OutcomeAllowed = "allowed"
	OutcomeError   = "error"

	// errorTypeRateLimit is the error.type of a failed take. The middleware is handed no error value,
	// so the type names the operation that failed rather than a Go type.
	errorTypeRateLimit = "rate_limit_error"
)

func init() {
	metric.RegisterHelp(metricNamespace, MetricNameRateLimitTake, "rate limiter token takes that completed, by outcome and error type")
	metric.RegisterHelp(metricNamespace, metricNameRateLimitAttemptedTake, "attempts to take a rate limiter token")
	metric.RegisterHelp(metricNamespace, metricNameRateLimitThrottledTakes, "rate limiter token takes throttled while waiting")
}

type metricMiddleware struct {
	metricWriter metric.Writer
}

func NewMetricMiddleware() Middleware {
	metricWriter := metric.NewWriter(metricNamespace)

	return NewMetricMiddlewareWithInterfaces(metricWriter)
}

func NewMetricMiddlewareWithInterfaces(metricWriter metric.Writer) *metricMiddleware {
	return &metricMiddleware{
		metricWriter: metricWriter,
	}
}

// OnTake records an attempt as soon as the limiter starts processing it.
func (m metricMiddleware) OnTake(ctx context.Context, i Invocation) {
	m.write(ctx, m.buildMetric(metricNameRateLimitAttemptedTake, i))
}

func (m metricMiddleware) OnRelease(ctx context.Context, i Invocation) {
	m.write(ctx, m.buildTake(i, OutcomeAllowed, metric.DimensionDefault))
}

func (m metricMiddleware) OnThrottle(ctx context.Context, i Invocation) {
	m.write(ctx, m.buildMetric(metricNameRateLimitThrottledTakes, i))
}

func (m metricMiddleware) OnError(ctx context.Context, i Invocation) {
	m.write(ctx, m.buildTake(i, OutcomeError, errorTypeRateLimit))
}

// buildTake reports one mutually exclusive terminal result of a take.
func (m metricMiddleware) buildTake(i Invocation, outcome string, errorType string) *metric.Datum {
	datum := m.buildMetric(MetricNameRateLimitTake, i)
	datum.Dimensions[dimensionOutcome] = outcome
	datum.Dimensions[metric.DimensionErrorType] = errorType

	return datum
}

func (m metricMiddleware) write(ctx context.Context, metric *metric.Datum) {
	m.metricWriter.WriteOne(ctx, metric)
}

func (m metricMiddleware) buildMetric(metricName string, i Invocation) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  clock.Provider.Now(),
		MetricName: metricName,
		Dimensions: metric.Dimensions{
			"name":   i.GetName(),
			"prefix": i.GetPrefix(),
		},
		Value: 1,
		Unit:  metric.UnitCount,
		Kind:  metric.KindCounter.Build(),
	}
}
