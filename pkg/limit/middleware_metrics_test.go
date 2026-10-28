package limit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestMetricMiddlewareCountsThrottledAttemptOnceAtCompletion(t *testing.T) {
	registry := prometheus.NewRegistry()
	prometheusWriter := metric.NewPrometheusWriterWithInterfaces(log.NewLogger(), registry, 1000, time.Second)
	var terminalInputs []terminalTakeObservation
	metricWriter := namespaceMetricWriter{
		namespace: metricNamespace,
		delegate:  prometheusWriter,
		observe: func(batch metric.Data) {
			for _, datum := range batch {
				if datum.MetricName == MetricNameRateLimitTake {
					terminalInputs = append(terminalInputs, terminalTakeObservation{
						outcome:   datum.Dimensions[dimensionOutcome],
						errorType: datum.Dimensions[metric.DimensionErrorType],
					})
				}
			}
		},
	}
	middlewareFactory := func() Middleware {
		return NewMetricMiddlewareWithInterfaces(metricWriter)
	}

	fakeClock := clock.NewFakeClock()
	builder, err := newInvocationBuilder("orders")
	require.NoError(t, err)
	limiter := NewFixedWindowLimiter(
		testIncrementer{
			increment: 2,
			ttl:       time.Second,
		},
		fakeClock,
		FixedWindowConfig{
			Name:   "orders",
			Cap:    1,
			Window: time.Minute,
		},
		builder,
	)
	limiter.WithMiddleware(middlewareFactory)

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- limiter.Wait(t.Context(), "tenant-a")
	}()
	fakeClock.BlockUntilTimers(1)
	fakeClock.Advance(time.Second)
	require.NoError(t, <-waitDone)

	errorBuilder, err := newInvocationBuilder("orders")
	require.NoError(t, err)
	failingLimiter := NewFixedWindowLimiter(
		testIncrementer{err: errors.New("increment failed")},
		fakeClock,
		FixedWindowConfig{
			Name:   "orders",
			Cap:    1,
			Window: time.Minute,
		},
		errorBuilder,
	)
	failingLimiter.WithMiddleware(middlewareFactory)
	require.Error(t, failingLimiter.Wait(t.Context(), "tenant-a"))
	require.Equal(t, []terminalTakeObservation{
		{
			outcome:   OutcomeAllowed,
			errorType: metric.DimensionDefault,
		},
		{
			outcome:   OutcomeError,
			errorType: errorTypeRateLimit,
		},
	}, terminalInputs)

	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 3)
	byName := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		byName[family.GetName()] = family
		require.Equal(t, dto.MetricType_COUNTER, family.GetType())
	}

	baseLabels := map[string]string{"name": "orders", "prefix": "tenant-a"}
	attempted := requireLimiterMetricFamily(t, byName, "gosoline_limit_attempted_takes_total")
	require.Len(t, attempted.Metric, 1)
	require.Equal(t, 2.0, attempted.Metric[0].GetCounter().GetValue())
	require.Equal(t, baseLabels, limiterMetricLabels(attempted.Metric[0]))

	throttled := requireLimiterMetricFamily(t, byName, "gosoline_limit_throttled_takes_total")
	require.Len(t, throttled.Metric, 1)
	require.Equal(t, 1.0, throttled.Metric[0].GetCounter().GetValue())
	require.Equal(t, baseLabels, limiterMetricLabels(throttled.Metric[0]))

	terminal := requireLimiterMetricFamily(t, byName, "gosoline_limit_takes_total")
	require.Len(t, terminal.Metric, 2)
	var terminalTotal float64
	foundAllowed := false
	foundError := false
	for _, sample := range terminal.Metric {
		labels := limiterMetricLabels(sample)
		require.Len(t, labels, 4)
		require.NotContains(t, labels, "trace_id")
		require.Equal(t, "orders", labels["name"])
		require.Equal(t, "tenant-a", labels["prefix"])
		terminalTotal += sample.GetCounter().GetValue()

		switch labels[dimensionOutcome] {
		case OutcomeAllowed:
			foundAllowed = true
			require.Empty(t, labels["error_type"])

			require.Equal(t, 1.0, sample.GetCounter().GetValue())
		case OutcomeError:
			foundError = true
			require.Equal(t, errorTypeRateLimit, labels["error_type"])
			require.Equal(t, 1.0, sample.GetCounter().GetValue())
		default:
			require.FailNow(t, "unexpected terminal take outcome", "%q", labels[dimensionOutcome])
		}
	}

	require.True(t, foundAllowed)
	require.True(t, foundError)
	require.Equal(t, 2.0, terminalTotal)
}

type terminalTakeObservation struct {
	outcome   string
	errorType string
}

type testIncrementer struct {
	increment int
	ttl       time.Duration
	err       error
}

func (b testIncrementer) Increment(context.Context, string) (*int, *time.Duration, error) {
	return &b.increment, &b.ttl, b.err
}

type namespaceMetricWriter struct {
	namespace string
	delegate  metric.Writer
	observe   func(metric.Data)
}

func (w namespaceMetricWriter) GetPriority() int {
	return w.delegate.GetPriority()
}

func (w namespaceMetricWriter) Write(ctx context.Context, batch metric.Data) {
	for _, datum := range batch {
		if datum.Namespace == "" {
			datum.Namespace = w.namespace
		}
	}

	if w.observe != nil {
		w.observe(batch)
	}
	w.delegate.Write(ctx, batch)
}

func (w namespaceMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
}

func requireLimiterMetricFamily(t *testing.T, families map[string]*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()
	family, ok := families[name]
	require.Truef(t, ok, "metric family %q was not exported", name)

	return family
}

func limiterMetricLabels(sample *dto.Metric) map[string]string {
	labels := make(map[string]string, len(sample.Label))
	for _, label := range sample.Label {
		labels[label.GetName()] = label.GetValue()
	}

	return labels
}
