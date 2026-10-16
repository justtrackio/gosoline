package metric_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

const writeGraceTime = 10 * time.Second

func Test_promWriter_WriteOne(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	tests := []struct {
		name         string
		data         *metric.Datum
		expectedName string
	}{
		{
			name: "no dimensions counter",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "counter",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitCount,
			},
			expectedName: "gosoline_counter_total",
		},
		{
			name: "no dimensions counter via kind",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "counter",
				Dimensions: nil,
				Value:      1,
				Kind:       metric.KindCounter.Build(),
			},
			expectedName: "gosoline_counter_total",
		},
		{
			name: "no dimensions gauge",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "gauge",
				Dimensions: nil,
				Value:      1,
			},
			expectedName: "gosoline_gauge",
		},
		{
			name: "no dimensions gauge",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "gauge",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitSeconds,
				Kind:       metric.KindGauge.Build(),
			},
			expectedName: "gosoline_gauge_seconds",
		},
		{
			name: "no dimensions histogram",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "histogram",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitSeconds,
				Kind:       metric.KindHistogram.Build(),
			},
			expectedName: "gosoline_histogram_seconds",
		},
		{
			name: "no dimensions summary",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "summary",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitSeconds,
				Kind:       metric.KindSummary.Build(),
			},
			expectedName: "gosoline_summary_seconds",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)
			w.WriteOne(t.Context(), tt.data)

			count, err := testutil.GatherAndCount(registry, tt.expectedName)
			assert.Equal(t, 1, count)
			assert.NoError(t, err)
		})
	}
}

func Test_promWriter_Write(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	type fields struct {
		unit  string
		name  string
		count int
	}

	tests := []struct {
		name     string
		initFunc func()
		data     metric.Data
		expected fields
	}{
		{
			name: "multiple metrics",
			data: metric.Data{
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "counter",
					Dimensions: nil,
					Value:      1,
					Unit:       metric.UnitCount,
				},
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "counter",
					Dimensions: nil,
					Value:      1,
					Unit:       metric.UnitCount,
				},
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "counter",
					Dimensions: nil,
					Value:      1,
					Unit:       metric.UnitCount,
				},
			},
			expected: fields{
				unit:  "Count",
				name:  "gosoline_counter_total",
				count: 3,
			},
		},
		{
			name: "multiple with default",
			initFunc: func() {
				metric.NewWriter("", &metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "counter",
					Value:      0,
					Unit:       metric.UnitCount,
				})
			},
			data: metric.Data{
				&metric.Datum{
					MetricName: "counter",
					Value:      1,
				},
				&metric.Datum{
					MetricName: "counter",
					Value:      1,
				},
				&metric.Datum{
					MetricName: "counter",
					Value:      1,
				},
			},
			expected: fields{
				unit:  "Count",
				name:  "gosoline_counter_total",
				count: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.initFunc != nil {
				tt.initFunc()
			}

			registry := prometheus.NewRegistry()
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)
			w.Write(t.Context(), tt.data)

			metricOutput := fmt.Sprintf(`
				# HELP %s unit: %s
				# TYPE %s counter
				%s %d
			`, tt.expected.name, tt.expected.unit, tt.expected.name, tt.expected.name, tt.expected.count)

			err := testutil.GatherAndCompare(registry, strings.NewReader(metricOutput), tt.expected.name)
			assert.NoError(t, err)
		})
	}
}

func Test_promWriter_ExceedsLimit(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	registry := prometheus.NewRegistry()
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1, writeGraceTime)
	w.WriteOne(t.Context(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "counter",
		Dimensions: nil,
		Value:      1,
		Unit:       metric.UnitCount,
	})

	w.WriteOne(t.Context(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "over_limit",
		Dimensions: nil,
		Value:      1,
		Unit:       metric.UnitCount,
	})

	count, err := testutil.GatherAndCount(registry, "gosoline_counter_total")
	assert.Equal(t, 1, count)
	assert.NoError(t, err)

	count, err = testutil.GatherAndCount(registry, "gosoline_over_limit_total")
	assert.Equal(t, 0, count)
	assert.NoError(t, err)
}

func Test_promWriter_Write_WithCanceledContextStillWrites(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	registry := prometheus.NewRegistry()
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	w.Write(ctx, metric.Data{
		&metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: "counter",
			Dimensions: nil,
			Value:      1,
			Unit:       metric.UnitCount,
		},
	})

	count, err := testutil.GatherAndCount(registry, "gosoline_counter_total")
	assert.NoError(t, err)
	assert.Equal(t, 1, count)
}

// Test_promWriter_RendersCanonicalNames pins down the prometheus rendering of a canonical namespace
// and leaf: the namespace becomes the subsystem, the leaf the name, and both carry the base unit and
// counter suffixes the prometheus convention requires.
func Test_promWriter_RendersCanonicalNames(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	tests := map[string]struct {
		datum        *metric.Datum
		expectedName string
	}{
		"duration": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "http.server",
				MetricName: "request.duration",
				Unit:       metric.UnitMilliseconds,
				Value:      250,
				Kind:       metric.KindHistogram.Build(),
			},
			expectedName: "gosoline_http_server_request_duration_seconds",
		},
		"counter": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "stream.consumer",
				MetricName: "errors",
				Unit:       metric.UnitCount,
				Value:      1,
				Kind:       metric.KindCounter.Build(),
			},
			expectedName: "gosoline_stream_consumer_errors_total",
		},
		"byte count": {
			datum: &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "kafka.broker",
				MetricName: "produce.batch.size",
				Unit:       metric.UnitBytes,
				Value:      2048,
				Kind:       metric.KindHistogram.Build(),
			},
			expectedName: "gosoline_kafka_broker_produce_batch_size_bytes",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)
			w.WriteOne(t.Context(), tt.datum)

			count, err := testutil.GatherAndCount(registry, tt.expectedName)
			assert.NoError(t, err)
			assert.Equal(t, 1, count, "expected %s to be exported", tt.expectedName)
		})
	}
}

// Test_promWriter_ScalesToBaseUnits pins down that a duration recorded in milliseconds is exported in
// seconds, the base unit prometheus requires.
func Test_promWriter_ScalesToBaseUnits(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	registry := prometheus.NewRegistry()
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)

	w.WriteOne(t.Context(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  "conc.scheduler",
		MetricName: "task.delay",
		Unit:       metric.UnitMillisecondsAverage,
		Value:      250,
		Kind:       metric.KindGauge.Build(),
	})

	expected := `
		# HELP gosoline_conc_scheduler_task_delay_seconds unit: UnitMillisecondsAverage
		# TYPE gosoline_conc_scheduler_task_delay_seconds gauge
		gosoline_conc_scheduler_task_delay_seconds 0.25
	`

	assert.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(expected), "gosoline_conc_scheduler_task_delay_seconds"))
}

// Test_promWriter_ClassifiesTimeBasedUnitsAsHistograms pins down that a time based unit without a
// declared instrument type is registered as a histogram rather than a summary or a gauge.
func Test_promWriter_ClassifiesTimeBasedUnitsAsHistograms(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	units := map[string]metric.StandardUnit{
		"milliseconds":         metric.UnitMilliseconds,
		"seconds":              metric.UnitSeconds,
		"milliseconds average": metric.UnitMillisecondsAverage,
		"seconds maximum":      metric.UnitSecondsMaximum,
	}

	for name, unit := range units {
		t.Run(name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)

			w.WriteOne(t.Context(), &metric.Datum{
				Priority:   metric.PriorityHigh,
				Namespace:  "http.client",
				MetricName: "request.duration",
				Unit:       unit,
				Value:      1,
			})

			families, err := registry.Gather()
			assert.NoError(t, err)
			assert.Len(t, families, 1)
			assert.Equal(t, dto.MetricType_HISTOGRAM, families[0].GetType())
		})
	}
}

// Test_promWriter_RendersDottedDimensionKeys pins down that a canonical dimension key survives into
// prometheus: a dot is not a valid character in a prometheus label name, so the key is rendered the
// same way the metric name is. Without this the whole datum is rejected at registration and the metric
// never appears.
func Test_promWriter_RendersDottedDimensionKeys(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	registry := prometheus.NewRegistry()
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)

	w.WriteOne(t.Context(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  "http.server",
		MetricName: "rejected.requests",
		Dimensions: metric.Dimensions{
			"http.server.name":    "api",
			"http.route":          "/users",
			"http.request.method": "GET",
		},
		Unit:  metric.UnitCount,
		Value: 1,
		Kind:  metric.KindCounter.Build(),
	})

	expected := `
		# HELP gosoline_http_server_rejected_requests_total unit: Count
		# TYPE gosoline_http_server_rejected_requests_total counter
		gosoline_http_server_rejected_requests_total{http_request_method="GET",http_route="/users",http_server_name="api"} 1
	`

	assert.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(expected), "gosoline_http_server_rejected_requests_total"))
}

func Test_promWriter_UsesCanonicalDefaultsBeforeNormalization(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	defaultDimensions := metric.Dimensions{
		"consumer.name": "batch",
		"error.type":    metric.DimensionDefault,
	}
	metric.NewWriter("stream", &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "consumed.messages",
		Dimensions: defaultDimensions,
		Unit:       metric.UnitCount,
		Kind:       metric.KindCounter.Build(),
	})

	inputDimensions := metric.Dimensions{
		"consumer.name": "batch",
		"error.type":    metric.DimensionDefault,
	}
	input := &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  "stream",
		MetricName: "consumed.messages",
		Dimensions: inputDimensions,
		Value:      3,
	}
	registry := prometheus.NewRegistry()
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)
	w.WriteOne(t.Context(), input)

	families, err := registry.Gather()
	assert.NoError(t, err)
	if !assert.Len(t, families, 1) {
		return
	}

	family := families[0]
	assert.Equal(t, "gosoline_stream_consumed_messages_total", family.GetName())
	assert.Equal(t, dto.MetricType_COUNTER, family.GetType())
	if !assert.Len(t, family.Metric, 1) {
		return
	}

	labels := make(map[string]string, len(family.Metric[0].Label))
	for _, label := range family.Metric[0].Label {
		labels[label.GetName()] = label.GetValue()
	}
	assert.Equal(t, "", labels["error_type"])
	assert.Equal(t, 3.0, family.Metric[0].GetCounter().GetValue())
	assert.Empty(t, input.Unit)
	assert.Equal(t, metric.Kind{}, input.Kind)
	assert.Equal(t, inputDimensions, input.Dimensions)
}

func Test_promWriter_ScalesHistogramBucketsToBaseUnits(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	buckets := []float64{1, 5, 10}
	registry := prometheus.NewRegistry()
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, writeGraceTime)
	w.WriteOne(t.Context(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  "http.server",
		MetricName: "request.duration",
		Unit:       metric.UnitMilliseconds,
		Value:      7,
		Kind:       metric.KindHistogram.WithBuckets(buckets).Build(),
	})

	families, err := registry.Gather()
	assert.NoError(t, err)
	if !assert.Len(t, families, 1) {
		return
	}

	family := families[0]
	assert.Equal(t, "gosoline_http_server_request_duration_seconds", family.GetName())
	assert.Equal(t, dto.MetricType_HISTOGRAM, family.GetType())
	if !assert.Len(t, family.Metric, 1) {
		return
	}

	histogram := family.Metric[0].GetHistogram()
	if !assert.NotNil(t, histogram) {
		return
	}
	assert.Equal(t, uint64(1), histogram.GetSampleCount())
	assert.InDelta(t, 0.007, histogram.GetSampleSum(), 1e-9)
	assert.Equal(t, []float64{1, 5, 10}, buckets)
	if !assert.Len(t, histogram.Bucket, 3) {
		return
	}

	expectedBounds := []float64{0.001, 0.005, 0.01}
	expectedCounts := []uint64{0, 0, 1}
	for i, bucket := range histogram.Bucket {
		assert.InDelta(t, expectedBounds[i], bucket.GetUpperBound(), 1e-12)
		assert.Equal(t, expectedCounts[i], bucket.GetCumulativeCount())
	}
}
