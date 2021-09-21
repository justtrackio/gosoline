package tracing_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
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
