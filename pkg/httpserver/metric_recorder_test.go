package httpserver

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServerMetricRecorder_ConcurrentRequests(t *testing.T) {
	writer := metricMocks.NewWriter(t)
	expectWriteOne(writer, MetricHttpConcurrentRequests, []float64{1, 2, 1, 0})
	recorder := newServerMetricRecorderWithInterfaces("api", clock.NewFakeClock(), writer, time.Hour)

	recorder.TrackRequestStarted(t.Context())
	recorder.TrackRequestStarted(t.Context())
	recorder.TrackRequestCompleted(t.Context())
	recorder.TrackRequestCompleted(t.Context())
}

func TestServerMetricRecorder_OpenConnections(t *testing.T) {
	writer := metricMocks.NewWriter(t)
	expectWriteOne(writer, MetricHttpOpenConnections, []float64{1, 2, 1, 0})
	recorder := newServerMetricRecorderWithInterfaces("api", clock.NewFakeClock(), writer, time.Hour)

	recorder.TrackConnectionOpened(t.Context())
	recorder.TrackConnectionOpened(t.Context())
	recorder.TrackConnectionClosed(t.Context())
	recorder.TrackConnectionClosed(t.Context())
}

func TestServerMetricRecorder_RunSamplesCurrentValues(t *testing.T) {
	writer := metricMocks.NewWriter(t)
	writer.EXPECT().WriteOne(matcher.Context, matchMetricDatum(MetricHttpConcurrentRequests, 1)).Return()
	writer.EXPECT().WriteOne(matcher.Context, matchMetricDatum(MetricHttpOpenConnections, 1)).Return()
	writer.EXPECT().Write(matcher.Context, matchMetricData(t, map[string]float64{
		MetricHttpConcurrentRequests: 1,
		MetricHttpOpenConnections:    1,
	})).Return().Twice()

	recorder := newServerMetricRecorderWithInterfaces("api", clock.NewFakeClock(), writer, time.Hour)
	recorder.TrackRequestStarted(t.Context())
	recorder.TrackConnectionOpened(t.Context())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	assert.NoError(t, recorder.Run(ctx))
}

func expectWriteOne(writer *metricMocks.Writer, metricName string, values []float64) {
	for _, value := range values {
		writer.EXPECT().WriteOne(matcher.Context, matchMetricDatum(metricName, value)).Return().Once()
	}
}

func matchMetricDatum(metricName string, value float64) any {
	return mock.MatchedBy(func(datum *metric.Datum) bool {
		return isMetricDatum(datum, metricName, value)
	})
}

func matchMetricData(t *testing.T, valuesByMetricName map[string]float64) any {
	return mock.MatchedBy(func(data metric.Data) bool {
		if !assert.Len(t, data, len(valuesByMetricName)) {
			return true
		}

		for _, datum := range data {
			value, ok := valuesByMetricName[datum.MetricName]
			assert.True(t, ok, "unexpected metric %s", datum.MetricName)
			if ok {
				assert.True(t, isMetricDatum(datum, datum.MetricName, value))
			}
		}

		return true
	})
}

func TestGetMetricRecorderDefaults(t *testing.T) {
	defaults := getMetricRecorderDefaults("api")

	assert.Len(t, defaults, 2)
	for _, datum := range defaults {
		assertMetricDatum(t, datum, datum.MetricName, 0)
	}
}

func isMetricDatum(datum *metric.Datum, metricName string, value float64) bool {
	if datum == nil {
		return false
	}

	return datum.Priority == metric.PriorityHigh &&
		datum.MetricName == metricName &&
		datum.Dimensions["ServerName"] == "api" &&
		len(datum.Dimensions) == 1 &&
		datum.Unit == metric.UnitCountMaximum &&
		reflect.DeepEqual(datum.Kind, metric.KindGauge.Build()) &&
		datum.Value == value
}

func assertMetricDatum(t *testing.T, datum *metric.Datum, metricName string, value float64) {
	assert.Equal(t, metric.PriorityHigh, datum.Priority)
	assert.Equal(t, metricName, datum.MetricName)
	assert.Equal(t, metric.Dimensions{"ServerName": "api"}, datum.Dimensions)
	assert.Equal(t, metric.UnitCountMaximum, datum.Unit)
	assert.Equal(t, metric.KindGauge.Build(), datum.Kind)
	assert.Equal(t, value, datum.Value)
}
