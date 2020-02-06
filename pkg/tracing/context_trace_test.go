package tracing_test

import (
	"context"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContextWithTrace(t *testing.T) {
	ctx := context.Background()
	trace := &tracing.Trace{
		TraceId:  "1-5e3d5273-7f0bd984ad68e2d290caeb84",
		Id:       "b1e67e41debe0b65",
		ParentId: "af260ec1430b6f17",
		Sampled:  true,
	}

	ctx = tracing.ContextWithTrace(ctx, trace)
	traceFromCtx := tracing.GetTraceFromContext(ctx)

	assert.NotNil(t, traceFromCtx)
	assert.Equal(t, trace, traceFromCtx)
}

func TestMessageWithTraceEncoder_Encode(t *testing.T) {
	tracer := getTracer()
	encoder := tracing.NewMessageWithTraceEncoder()

	ctx, span := tracer.StartSpan("test-span")
	defer span.Finish()

	attributes, err := encoder.Encode(ctx, map[string]interface{}{})

	assert.NoError(t, err)
	assert.Contains(t, attributes, "traceId")
	assert.Regexp(t, "Root=[^;]+;Parent=[^;]+;Sampled=[01]", attributes["traceId"])
}

func TestMessageWithTraceEncoder_Decode(t *testing.T) {
	ctx := context.Background()
	attributes := map[string]interface{}{
		"traceId": "Root=1-5e3d557d-d06c248cc50169bd71b44fec;Parent=af297a5da6453826;Sampled=1",
	}

	encoder := tracing.NewMessageWithTraceEncoder()
	ctx, err := encoder.Decode(ctx, attributes)

	trace := tracing.GetTraceFromContext(ctx)
	expected := &tracing.Trace{
		TraceId:  "1-5e3d557d-d06c248cc50169bd71b44fec",
		Id:       "",
		ParentId: "af297a5da6453826",
		Sampled:  true,
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, trace)
}
