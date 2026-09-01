package metric_test

import (
	"testing"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestOtelWriterHistogramBuckets(t *testing.T) {
	defaultBuckets := []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000}

	tests := map[string]struct {
		buckets        []float64
		expectedBounds []float64
	}{
		"custom buckets": {
			buckets:        []float64{1, 5, 10},
			expectedBounds: []float64{1, 5, 10},
		},
		"nil buckets": {
			expectedBounds: defaultBuckets,
		},
		"empty buckets": {
			buckets:        []float64{},
			expectedBounds: defaultBuckets,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			reader := sdkmetric.NewManualReader()
			provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() {
				require.NoError(t, provider.Shutdown(t.Context()))
			})

			logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
			writer := metric.NewOtelWriterWithInterfaces(logger, provider.Meter("test"))
			writer.WriteOne(t.Context(), &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "RequestDuration",
				Unit:       metric.UnitMilliseconds,
				Value:      7,
				Kind:       metric.KindHistogram.WithBuckets(tt.buckets).Build(),
			})

			var exported metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(t.Context(), &exported))
			require.Len(t, exported.ScopeMetrics, 1)
			require.Len(t, exported.ScopeMetrics[0].Metrics, 1)

			histogram, ok := exported.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[float64])
			require.True(t, ok)
			require.Len(t, histogram.DataPoints, 1)
			assert.Equal(t, tt.expectedBounds, histogram.DataPoints[0].Bounds)
		})
	}
}

// TestOtelWriterRendersCanonicalNames pins down the otel rendering of a canonical namespace and leaf:
// the dotted canonical form, prefixed with `gosoline.` unless a semantic convention owns the metric,
// and the unit on the instrument rather than in the name.
func TestOtelWriterRendersCanonicalNames(t *testing.T) {
	tests := map[string]struct {
		datum        *metric.Datum
		expectedName string
		expectedUnit string
	}{
		"semantic convention metric": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "http.server",
				MetricName: "request.duration",
				Unit:       metric.UnitMilliseconds,
				Value:      250,
				Kind:       metric.KindHistogram.Build(),
			},
			expectedName: "http.server.request.duration",
			expectedUnit: "s",
		},
		"gosoline specific metric": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "stream.consumer",
				MetricName: "errors",
				Unit:       metric.UnitCount,
				Value:      1,
				Kind:       metric.KindCounter.Build(),
			},
			expectedName: "gosoline.stream.consumer.errors",
			expectedUnit: "{error}",
		},
		"gosoline metric inside a semantic convention namespace": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "http.server",
				MetricName: "rejected.requests",
				Unit:       metric.UnitCount,
				Value:      1,
				Kind:       metric.KindCounter.Build(),
			},
			expectedName: "gosoline.http.server.rejected.requests",
			expectedUnit: "{request}",
		},
		"custom aggregation unit resolves to its base unit": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "conc.scheduler",
				MetricName: "task.delay",
				Unit:       metric.UnitMillisecondsAverage,
				Value:      250,
				Kind:       metric.KindHistogram.Build(),
			},
			expectedName: "gosoline.conc.scheduler.task.delay",
			expectedUnit: "s",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			exported := writeAndCollect(t, tt.datum)

			require.Len(t, exported.ScopeMetrics, 1)
			require.Len(t, exported.ScopeMetrics[0].Metrics, 1)

			assert.Equal(t, tt.expectedName, exported.ScopeMetrics[0].Metrics[0].Name)
			assert.Equal(t, tt.expectedUnit, exported.ScopeMetrics[0].Metrics[0].Unit)
			assert.NotContains(t, exported.ScopeMetrics[0].Metrics[0].Name, "_total")
			assert.NotContains(t, exported.ScopeMetrics[0].Metrics[0].Name, "_seconds")
		})
	}
}

// TestOtelWriterScalesToBaseUnits pins down that a duration recorded in milliseconds is exported in
// seconds.
func TestOtelWriterScalesToBaseUnits(t *testing.T) {
	exported := writeAndCollect(t, &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  "http.server",
		MetricName: "request.duration",
		Unit:       metric.UnitMilliseconds,
		Value:      250,
		Kind:       metric.KindHistogram.Build(),
	})

	require.Len(t, exported.ScopeMetrics, 1)
	require.Len(t, exported.ScopeMetrics[0].Metrics, 1)

	histogram, ok := exported.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[float64])
	require.True(t, ok)
	require.Len(t, histogram.DataPoints, 1)
	assert.InDelta(t, 0.25, histogram.DataPoints[0].Sum, 1e-9)
}

func writeAndCollect(t *testing.T, datum *metric.Datum) metricdata.ResourceMetrics {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(t.Context()))
	})

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	writer := metric.NewOtelWriterWithInterfaces(logger, provider.Meter("test"))
	writer.WriteOne(t.Context(), datum)

	var exported metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &exported))

	return exported
}
