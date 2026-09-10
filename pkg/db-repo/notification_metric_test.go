package db_repo

import (
	"context"
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifierWritesSuccessAsDefaultErrorOutcome(t *testing.T) {
	writer := notificationMetricWriter{datums: make(chan *metric.Datum, 1)}
	notifier := notifier{
		metric:  writer,
		modelId: mdl.ModelId{Name: "model"},
	}

	notifier.writeMetric(t.Context(), nil)

	assertNotificationOutcome(t, <-writer.datums, metric.DimensionDefault)
}

func TestNotifierWritesFailureAsErrorTypeOutcome(t *testing.T) {
	writer := notificationMetricWriter{datums: make(chan *metric.Datum, 1)}
	notifier := notifier{
		metric:  writer,
		modelId: mdl.ModelId{Name: "model"},
	}
	err := errors.New("publish failed")

	notifier.writeMetric(t.Context(), err)

	assertNotificationOutcome(t, <-writer.datums, metric.ErrorType(err))
}

func TestDefaultNotifierMetricsContainsSuccessOutcome(t *testing.T) {
	defaults := getDefaultNotifierMetrics(mdl.ModelId{Name: "model"})

	require.Len(t, defaults, 1)
	assertNotificationOutcome(t, defaults[0], metric.DimensionDefault)
	assert.Zero(t, defaults[0].Value)
}

func assertNotificationOutcome(t *testing.T, datum *metric.Datum, errorType string) {
	t.Helper()
	assert.Equal(t, metricNameNotifications, datum.MetricName)
	assert.Equal(t, errorType, datum.Dimensions[metric.DimensionErrorType])
	assert.Equal(t, metric.UnitCount, datum.Unit)
	assert.Equal(t, metric.KindCounter.Build(), datum.Kind)
}

type notificationMetricWriter struct {
	datums chan *metric.Datum
}

func (w notificationMetricWriter) GetPriority() int {
	return metric.PriorityLow
}

func (w notificationMetricWriter) Write(ctx context.Context, data metric.Data) {
	for _, datum := range data {
		w.WriteOne(ctx, datum)
	}
}

func (w notificationMetricWriter) WriteOne(_ context.Context, datum *metric.Datum) {
	w.datums <- datum
}
