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
