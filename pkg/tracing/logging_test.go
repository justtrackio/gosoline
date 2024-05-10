package tracing_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/justtrackio/gosoline/pkg/tracing/mocks"
	"github.com/stretchr/testify/suite"
)

type LoggingSuite struct {
	suite.Suite

	ctx  context.Context
	span *mocks.Span
}

func (s *LoggingSuite) SetupTest() {
	s.span = mocks.NewSpan(s.T())

	ctx := s.T().Context()
	s.ctx = tracing.ContextWithSpan(ctx, s.span)
}

func (s *LoggingSuite) TestContextTraceFieldsResolver_FromSpan() {
	trace := &tracing.Trace{
		TraceId:  "1-5e3d5273-7f0bd984ad68e2d290caeb84",
		Id:       "b1e67e41debe0b65",
		ParentId: "af260ec1430b6f17",
		Sampled:  true,
	}
	s.span.EXPECT().GetTrace().Return(trace).Once()

	fields := tracing.ContextTraceFieldsResolver(s.ctx)

	s.Contains(fields, "trace_id")
	s.Equal("1-5e3d5273-7f0bd984ad68e2d290caeb84", fields["trace_id"])
}

func (s *LoggingSuite) TestContextTraceFieldsResolver_FromContextTrace_EmptySpan() {
	trace := &tracing.Trace{
		TraceId:  "goso:83268bc4-dc7a-4674-8e71-d8b6e0e01543",
		Id:       "00000000-0000-0000-0000-000000000000",
		ParentId: "00000000-0000-0000-0000-000000000000",
		Sampled:  false,
	}
	s.span.EXPECT().GetTrace().Return(nil).Once()
	ctx := tracing.ContextWithTrace(s.ctx, trace)

	fields := tracing.ContextTraceFieldsResolver(ctx)

	s.Contains(fields, "trace_id")
	s.Equal("goso:83268bc4-dc7a-4674-8e71-d8b6e0e01543", fields["trace_id"])
}

func (s *LoggingSuite) TestContextTraceFieldsResolver_FromContextTrace() {
	trace := &tracing.Trace{
		TraceId:  "goso:14406634-410c-42a6-9247-411167cad1e4",
		Id:       "00000000-0000-0000-0000-000000000000",
		ParentId: "00000000-0000-0000-0000-000000000000",
		Sampled:  false,
	}
	ctx := tracing.ContextWithTrace(s.T().Context(), trace)

	fields := tracing.ContextTraceFieldsResolver(ctx)

	s.Contains(fields, "trace_id")
	s.Equal("goso:14406634-410c-42a6-9247-411167cad1e4", fields["trace_id"])
}

func (s *LoggingSuite) TestContextTraceFieldsResolver_EmptyContext() {
	fields := tracing.ContextTraceFieldsResolver(s.T().Context())

	s.NotContains(fields, "trace_id")
}

func (s *LoggingSuite) TestLoggerErrorHook() {
	errToLog := fmt.Errorf("unexpected error")
	s.span.EXPECT().AddError(errToLog).Once()

	hook := tracing.NewLoggerErrorHandler()
	err := hook.Log(time.Time{}, 0, "", []any{}, errToLog, log.Data{
		Context: s.ctx,
	})

	s.NoError(err)
}

func TestLoggingSuite(t *testing.T) {
	suite.Run(t, new(LoggingSuite))
}
