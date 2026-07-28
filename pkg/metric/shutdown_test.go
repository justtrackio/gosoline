package metric_test

import (
	"context"
	"errors"
	"testing"

	gosolineMetric "github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestShutdownHandler_NoProvider(t *testing.T) {
	err := (&gosolineMetric.ShutdownHandler{}).Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestShutdownHandler_CallsProvider(t *testing.T) {
	exporter := &shutdownExporter{}
	handler := &gosolineMetric.ShutdownHandler{}
	handler.AddProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter))))

	err := handler.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, exporter.called)
}

func TestShutdownHandler_PropagatesError(t *testing.T) {
	expected := errors.New("shutdown failed")
	handler := &gosolineMetric.ShutdownHandler{}
	handler.AddProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(&shutdownExporter{err: expected}))))

	err := handler.Shutdown(context.Background())
	assert.ErrorIs(t, err, expected)
}

type shutdownExporter struct {
	called bool
	err    error
}

func (e *shutdownExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *shutdownExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *shutdownExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	return nil
}

func (e *shutdownExporter) ForceFlush(context.Context) error {
	return nil
}

func (e *shutdownExporter) Shutdown(context.Context) error {
	e.called = true

	return e.err
}
