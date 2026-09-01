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
		name string
		data *metric.Datum
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
		},
		{
			name: "no dimensions gauge",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "gauge",
				Dimensions: nil,
				Value:      1,
			},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "ns:test", 1000, writeGraceTime)
			w.WriteOne(t.Context(), tt.data)

			count, err := testutil.GatherAndCount(registry, "ns:test_"+tt.data.MetricName)
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
				name:  "ns:test:write_counter",
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
				name:  "ns:test:write_counter",
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
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "ns:test:write", 1000, writeGraceTime)
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
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "ns:test:exceedslimit", 1, writeGraceTime)
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

	count, err := testutil.GatherAndCount(registry, "ns:test:exceedslimit_counter")
	assert.Equal(t, 1, count)
	assert.NoError(t, err)

	count, err = testutil.GatherAndCount(registry, "ns:test:exceedslimit_over_limit")
	assert.Equal(t, 0, count)
	assert.NoError(t, err)
}

func Test_promWriter_Write_WithCanceledContextStillWrites(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))

	registry := prometheus.NewRegistry()
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "ns:test:grace", 1000, writeGraceTime)

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

	count, err := testutil.GatherAndCount(registry, "ns:test:grace_counter")
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
			expectedName: "myapp_http_server_request_duration_seconds",
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
			expectedName: "myapp_stream_consumer_errors_total",
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
			expectedName: "myapp_kafka_broker_produce_batch_size_bytes",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "myapp", 1000, writeGraceTime)
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
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "myapp", 1000, writeGraceTime)

	w.WriteOne(t.Context(), &metric.Datum{
		Priority:   metric.PriorityHigh,
		Namespace:  "conc.scheduler",
		MetricName: "task.delay",
		Unit:       metric.UnitMillisecondsAverage,
		Value:      250,
		Kind:       metric.KindGauge.Build(),
	})

	expected := `
		# HELP myapp_conc_scheduler_task_delay_seconds unit: UnitMillisecondsAverage
		# TYPE myapp_conc_scheduler_task_delay_seconds gauge
		myapp_conc_scheduler_task_delay_seconds 0.25
	`

	assert.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(expected), "myapp_conc_scheduler_task_delay_seconds"))
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
			w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "myapp", 1000, writeGraceTime)

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
	w := metric.NewPrometheusWriterWithInterfaces(logger, registry, "myapp", 1000, writeGraceTime)

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
		# HELP myapp_http_server_rejected_requests_total unit: Count
		# TYPE myapp_http_server_rejected_requests_total counter
		myapp_http_server_rejected_requests_total{http_request_method="GET",http_route="/users",http_server_name="api"} 1
	`

	assert.NoError(t, testutil.GatherAndCompare(registry, strings.NewReader(expected), "myapp_http_server_rejected_requests_total"))
}
