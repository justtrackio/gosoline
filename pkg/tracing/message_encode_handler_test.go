package tracing_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

func TestMessageWithTraceEncoder_Encode(t *testing.T) {
	tracer := getTracer(t)
	encoder := tracing.NewMessageWithTraceEncoder(tracing.TraceIdErrorReturnStrategy{})

	ctx, span := tracer.StartSpan("test-span")
	defer span.Finish()

	_, attributes, err := encoder.Encode(ctx, nil, map[string]string{})

	assert.NoError(t, err)
	assert.Contains(t, attributes, "traceId")
	assert.Regexp(t, "Root=[^;]+;Parent=[^;]+;Sampled=[01]", attributes["traceId"])
}

func TestMessageWithTraceEncoder_Encode_TraceFromContext(t *testing.T) {
	encoder := tracing.NewMessageWithTraceEncoder(tracing.TraceIdErrorReturnStrategy{})

	trace := &tracing.Trace{
		TraceId:  "goso:14406634-410c-42a6-9247-411167cad1e4",
		Id:       "00000000-0000-0000-0000-000000000000",
		ParentId: "00000000-0000-0000-0000-000000000000",
		Sampled:  false,
	}
	ctx := tracing.ContextWithTrace(t.Context(), trace)
	_, attributes, err := encoder.Encode(ctx, nil, map[string]string{})

	assert.NoError(t, err)
	assert.Contains(t, attributes, "traceId")
	assert.Regexp(t, "Root=[^;]+;Parent=[^;]+;Sampled=[01]", attributes["traceId"])
}

func TestMessageWithTraceEncoder_Decode(t *testing.T) {
	ctx := t.Context()
	attributes := map[string]string{
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
	ctx := t.Context()
	attributes := map[string]string{
		"traceId": "1-5e3d557d-d06c248cc50169bd71b44fec",
	}

	logger := mocks.NewLogger(t)
	logger.EXPECT().WithFields(log.Fields{
		"stacktrace": "mocked trace",
	}).Return(logger).Once()
	logger.EXPECT().Warn(
		"trace id is invalid: %s",
		"the traceId attribute is invalid: the trace id [1-5e3d557d-d06c248cc50169bd71b44fec] should contain a root part",
	).Once()

	strategy := tracing.NewTraceIdErrorWarningStrategyWithInterfaces(logger, log.GetMockedStackTrace)

	encoder := tracing.NewMessageWithTraceEncoder(strategy)
	ctx, decodedAttributes, err := encoder.Decode(ctx, nil, attributes)

	trace := tracing.GetTraceFromContext(ctx)

	assert.NoError(t, err)
	assert.Contains(t, decodedAttributes, "traceId")
	assert.Nil(t, trace)
}
