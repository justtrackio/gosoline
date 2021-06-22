package tracing_test

import (
	"context"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMessageWithTraceEncoder_Encode(t *testing.T) {
	tracer := getTracer(t)
	encoder := tracing.NewMessageWithTraceEncoder(tracing.TraceIdErrorReturnStrategy{})

	ctx, span := tracer.StartSpan("test-span")
	defer span.Finish()

	_, attributes, err := encoder.Encode(ctx, nil, map[string]interface{}{})

	assert.NoError(t, err)
	assert.Contains(t, attributes, "traceId")
	assert.Regexp(t, "Root=[^;]+;Parent=[^;]+;Sampled=[01]", attributes["traceId"])
}

func TestMessageWithTraceEncoder_Decode(t *testing.T) {
	ctx := context.Background()
	attributes := map[string]interface{}{
		"traceId": "Root=1-5e3d557d-d06c248cc50169bd71b44fec;Parent=af297a5da6453826;Sampled=1",
	}

	encoder := tracing.NewMessageWithTraceEncoder(tracing.TraceIdErrorReturnStrategy{})
	ctx, decodedAttributes, err := encoder.Decode(ctx, nil, attributes)

	trace := tracing.GetTraceFromContext(ctx)
	expected := &tracing.Trace{
		TraceId:  "1-5e3d557d-d06c248cc50169bd71b44fec",
		Id:       "",
		ParentId: "af297a5da6453826",
		Sampled:  true,
	}

	assert.NoError(t, err)
	assert.NotContains(t, decodedAttributes, "traceId")
	assert.Equal(t, expected, trace)
}

func TestMessageWithTraceEncoder_Decode_Warning(t *testing.T) {
	ctx := context.Background()
	attributes := map[string]interface{}{
		"traceId": "1-5e3d557d-d06c248cc50169bd71b44fec",
	}

	logger := new(mocks.Logger)
	logger.On("WithFields", log.Fields{
		"stacktrace": "mocked trace",
	}).Return(logger).Once()
	logger.On("Warn", "trace id is invalid: %s", "the traceId attribute is invalid: the trace id [1-5e3d557d-d06c248cc50169bd71b44fec] should consist of at least 2 parts")

	strategy := tracing.NewTraceIdErrorWarningStrategyWithInterfaces(logger, log.GetMockedStackTrace)

	encoder := tracing.NewMessageWithTraceEncoder(strategy)
	ctx, decodedAttributes, err := encoder.Decode(ctx, nil, attributes)

	trace := tracing.GetTraceFromContext(ctx)

	assert.NoError(t, err)
	assert.Contains(t, decodedAttributes, "traceId")
	assert.Nil(t, trace)

	logger.AssertExpectations(t)
}
