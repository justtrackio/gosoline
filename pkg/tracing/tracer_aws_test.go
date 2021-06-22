package tracing_test

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAwsTracer_StartSubSpan(t *testing.T) {
	tracer := getTracer(t)

	ctx, trans := tracer.StartSpan("test_trans")
	_, span := tracer.StartSubSpan(ctx, "test_span")

	assert.Equal(t, trans.GetTrace().TraceId, span.GetTrace().TraceId, "the trace ids should match")
	assert.Equal(t, trans.GetTrace().Sampled, span.GetTrace().Sampled, "the sample decision should match")
	assert.NotEqual(t, trans.GetTrace().Id, span.GetTrace().Id, "the span ids should be different")
	assert.Empty(t, trans.GetTrace().GetParentId(), "the parent id of the transaction should be empty")
	assert.Empty(t, span.GetTrace().GetParentId(), "the parent id of the span should be empty")
}

func TestAwsTracer_StartSpanFromContextWithSpan(t *testing.T) {
	tracer := getTracer(t)

	ctx, transRoot := tracer.StartSpan("test_trans")
	_, transChild := tracer.StartSpanFromContext(ctx, "another_trace")

	assert.Equal(t, transRoot.GetTrace().TraceId, transChild.GetTrace().TraceId, "the trace ids should match")
	assert.Equal(t, transRoot.GetTrace().Sampled, transChild.GetTrace().Sampled, "the sample decision should match")
	assert.NotEqual(t, transRoot.GetTrace().Id, transChild.GetTrace().Id, "the span ids should be different")
	assert.Empty(t, transRoot.GetTrace().GetParentId(), "the parent id of the root transaction should be empty")
	assert.NotEmpty(t, transChild.GetTrace().GetParentId(), "the parent id of the child transaction should not be empty")
	assert.Equal(t, transRoot.GetTrace().Id, transChild.GetTrace().ParentId, "span id of root should match parent id of child")
}

func TestAwsTracer_StartSpanFromContextWithTrace(t *testing.T) {
	tracer := getTracer(t)

	trace := &tracing.Trace{
		TraceId:  "1-5759e988-bd862e3fe1be46a994272793",
		Id:       "54567a67e89cdf88",
		ParentId: "53995c3f42cd8ad8",
		Sampled:  true,
	}

	ctx := tracing.ContextWithTrace(context.Background(), trace)
	_, transChild := tracer.StartSpanFromContext(ctx, "another_trace")

	assert.Equal(t, trace.TraceId, transChild.GetTrace().TraceId, "the trace ids should match")
	assert.Equal(t, trace.Sampled, transChild.GetTrace().Sampled, "the sample decision should match")
	assert.NotEqual(t, trace.Id, transChild.GetTrace().Id, "the span ids should be different")
	assert.NotEmpty(t, transChild.GetTrace().GetParentId(), "the parent id of the child transaction should not be empty")
	assert.Equal(t, trace.ParentId, transChild.GetTrace().ParentId, "span id of root should match parent id of child")
}

func getTracer(t *testing.T) tracing.Tracer {
	logger := mocks.NewLoggerMockedAll()
	tracer, err := tracing.NewAwsTracerWithInterfaces(logger, cfg.AppId{
		Project:     "test_project",
		Environment: "test_env",
		Family:      "test_family",
		Application: "test_name",
	}, &tracing.XRaySettings{Enabled: true})

	assert.NoError(t, err, "we should be able to get a tracer")

	return tracer
}
