package tracing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestShutdownHandler_NoProvider(t *testing.T) {
	err := (&tracing.ShutdownHandler{}).Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestShutdownHandler_CallsProvider(t *testing.T) {
	exporter := &shutdownExporter{}
	handler := &tracing.ShutdownHandler{}
	handler.AddProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter)))

	err := handler.Shutdown(context.Background())
	assert.NoError(t, err)
	assert.True(t, exporter.called)
}

func TestShutdownHandler_PropagatesError(t *testing.T) {
	expected := errors.New("shutdown failed")
	handler := &tracing.ShutdownHandler{}
	handler.AddProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(&shutdownExporter{err: expected})))

	err := handler.Shutdown(context.Background())
	assert.ErrorIs(t, err, expected)
}

type shutdownExporter struct {
	called bool
	err    error
}

func (e *shutdownExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *shutdownExporter) Shutdown(context.Context) error {
	e.called = true

	return e.err
}
