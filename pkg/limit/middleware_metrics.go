package limit

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNamespace = "limit"

	MetricNameRateLimitTake = "takes"

	// dimensionOutcome tells apart how one take ended. A take is counted once, when it ends, so the
	// attempts are the sum over this attribute rather than a metric of their own.
	dimensionOutcome = "outcome"

	OutcomeAllowed   = "allowed"
	OutcomeThrottled = "throttled"
	OutcomeError     = "error"

	// errorTypeRateLimit is the error.type of a failed take. The middleware is handed no error value,
	// so the type names the operation that failed rather than a Go type.
	errorTypeRateLimit = "rate_limit_error"
)

func init() {
	metric.RegisterHelp(metricNamespace, MetricNameRateLimitTake, "attempts to take a rate limiter token, by outcome and error type")
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

// OnTake records nothing: a take is counted once, when it ends, so its outcome is known. The number of
// attempts is the sum of MetricNameRateLimitTake over every outcome.
func (m metricMiddleware) OnTake(context.Context, Invocation) {}

func (m metricMiddleware) OnRelease(ctx context.Context, i Invocation) {
	m.write(ctx, m.buildTake(i, OutcomeAllowed, metric.DimensionDefault))
}

func (m metricMiddleware) OnThrottle(ctx context.Context, i Invocation) {
	m.write(ctx, m.buildTake(i, OutcomeThrottled, metric.DimensionDefault))
}

func (m metricMiddleware) OnError(ctx context.Context, i Invocation) {
	m.write(ctx, m.buildTake(i, OutcomeError, errorTypeRateLimit))
}

// buildTake reports one completed attempt to take a token. A throttled or failed attempt is the same
// metric told apart by its outcome and error type, so neither needs a metric of its own.
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
