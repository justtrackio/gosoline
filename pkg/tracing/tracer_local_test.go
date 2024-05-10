package tracing_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestLocalTracer_StartSpan(t *testing.T) {
	tracer := tracing.NewLocalTracer()

	ctx1, _ := tracer.StartSpan("test_trans1")
	ctx2, _ := tracer.StartSpan("test_trans2")

	traceId1 := tracing.GetTraceIdFromContext(ctx1)
	traceId2 := tracing.GetTraceIdFromContext(ctx2)

	assert.NotNil(t, traceId1, "the trace id should exist")
	assert.NotNil(t, traceId2, "the trace id should exist")
	assert.NotEqual(t, "", traceId1, "the trace id should not be empty")
	assert.NotEqual(t, "", traceId2, "the trace id should not be empty")
	assert.NotEqual(t, *traceId1, *traceId2, "the trace ids should not match")
}

func TestLocalTracer_StartSpanFromContext(t *testing.T) {
	tracer := tracing.NewLocalTracer()

	ctx1, _ := tracer.StartSpanFromContext(t.Context(), "test_trans1")
	ctx2, _ := tracer.StartSpanFromContext(t.Context(), "test_trans2")

	traceId1 := tracing.GetTraceIdFromContext(ctx1)
	traceId2 := tracing.GetTraceIdFromContext(ctx2)

	assert.NotNil(t, traceId1, "the trace id should exist")
	assert.NotNil(t, traceId2, "the trace id should exist")
	assert.NotEqual(t, "", traceId1, "the trace id should not be empty")
	assert.NotEqual(t, "", traceId2, "the trace id should not be empty")
	assert.NotEqual(t, *traceId1, *traceId2, "the trace ids should not match")
}

func TestLocalTracer_StartSpanFromContextWithTrace(t *testing.T) {
	tracer := tracing.NewLocalTracer()

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
