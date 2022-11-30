package metric_test

import (
	"fmt"
	"strings"
	"testing"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func Test_promWriter_WriteOne(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()

	tests := []struct {
		name string
		data *metric.Datum
	}{
		{
			name: "no dimensions prom-counter",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "counter",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitPromCounter,
			},
		},
		{
			name: "no dimensions prom-gauge",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "gauge",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitPromGauge,
			},
		},
		{
			name: "no dimensions prom-histogram",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "histogram",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitPromHistogram,
			},
		},
		{
			name: "no dimensions prom-summary",
			data: &metric.Datum{
				Priority:   metric.PriorityHigh,
				MetricName: "summary",
				Dimensions: nil,
				Value:      1,
				Unit:       metric.UnitPromSummary,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			w := metric.NewPromWriterWithInterfaces(logger, registry, "ns:test", 1000)
			w.WriteOne(tt.data)

			count, err := testutil.GatherAndCount(registry, "ns:test_"+tt.data.MetricName)
			assert.Equal(t, 1, count)
			assert.NoError(t, err)
		})
	}
}

func Test_promWriter_Write(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()

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
					Unit:       metric.UnitPromCounter,
				},
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "counter",
					Dimensions: nil,
					Value:      1,
					Unit:       metric.UnitPromCounter,
				},
				&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "counter",
					Dimensions: nil,
					Value:      1,
					Unit:       metric.UnitPromCounter,
				},
			},
			expected: fields{
				unit:  "prom-counter",
				name:  "ns:test:write_counter",
				count: 3,
			},
		},
		{
			name: "multiple with default",
			initFunc: func() {
				metric.NewWriter(&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "counter",
					Value:      0,
					Unit:       metric.UnitPromCounter,
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
				unit:  "prom-counter",
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
			w := metric.NewPromWriterWithInterfaces(logger, registry, "ns:test:write", 1000)
			w.Write(tt.data)

			var metricOutput = fmt.Sprintf(`
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
	logger := logMocks.NewLoggerMockedAll()

	registry := prometheus.NewRegistry()
	w := metric.NewPromWriterWithInterfaces(logger, registry, "ns:test:exceedslimit", 1)
	w.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "counter",
		Dimensions: nil,
		Value:      1,
		Unit:       metric.UnitPromCounter,
	})

	w.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "over_limit",
		Dimensions: nil,
		Value:      1,
		Unit:       metric.UnitPromCounter,
	})

	count, err := testutil.GatherAndCount(registry, "ns:test:exceedslimit_counter")
	assert.Equal(t, 1, count)
	assert.NoError(t, err)

	count, err = testutil.GatherAndCount(registry, "ns:test:exceedslimit_over_limit")
	assert.Equal(t, 0, count)
	assert.NoError(t, err)
}
