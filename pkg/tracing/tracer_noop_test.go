package tracing_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestNoopTracer_StartSpan(t *testing.T) {
	tracer := tracing.NewNoopTracer()

	ctx, _ := tracer.StartSpan("test_trans")
	traceId := tracing.GetTraceIdFromContext(ctx)

	assert.Nil(t, traceId, "the trace id should not exist")
}

func TestNoopTracer_StartSpanFromContext(t *testing.T) {
	tracer := tracing.NewNoopTracer()

	ctx, _ := tracer.StartSpanFromContext(t.Context(), "test_trans")
	traceId := tracing.GetTraceIdFromContext(ctx)

	assert.Nil(t, traceId, "the trace id should not exist")
}

func TestNoopTracer_StartSpanFromContextWithTrace(t *testing.T) {
	tracer := tracing.NewNoopTracer()

	trace := &tracing.Trace{
		TraceId:  "1-5759e988-bd862e3fe1be46a994272793",
		Id:       "54567a67e89cdf88",
		ParentId: "53995c3f42cd8ad8",
		Sampled:  true,
	}

	ctx := tracing.ContextWithTrace(t.Context(), trace)
	ctx, _ = tracer.StartSpanFromContext(ctx, "another_trace")
	traceId := tracing.GetTraceIdFromContext(ctx)

	assert.NotNil(t, traceId, "the trace id should exist")
	assert.NotEqual(t, "", traceId, "the trace id should not be empty")
	assert.Equal(t, tracing.TraceToString(trace), *traceId, "the trace ids should equal the provided value")
}
