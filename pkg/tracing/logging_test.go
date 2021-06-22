package tracing_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/applike/gosoline/pkg/tracing/mocks"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type LoggingSuite struct {
	suite.Suite

	ctx  context.Context
	span *mocks.Span
}

func (s *LoggingSuite) SetupTest() {
	s.span = new(mocks.Span)

	ctx := context.Background()
	s.ctx = tracing.ContextWithSpan(ctx, s.span)
}

func (s *LoggingSuite) TestContextTraceFieldsResolver() {
	trace := &tracing.Trace{
		TraceId:  "1-5e3d5273-7f0bd984ad68e2d290caeb84",
		Id:       "b1e67e41debe0b65",
		ParentId: "af260ec1430b6f17",
		Sampled:  true,
	}
	s.span.On("GetTrace").Return(trace)

	fields := tracing.ContextTraceFieldsResolver(s.ctx)

	s.Contains(fields, "trace_id")
	s.Equal("1-5e3d5273-7f0bd984ad68e2d290caeb84", fields["trace_id"])
	s.span.AssertExpectations(s.T())
}

func (s *LoggingSuite) TestLoggerErrorHook() {
	errToLog := fmt.Errorf("unexpected error")
	s.span.On("AddError", errToLog)

	hook := tracing.NewLoggerErrorHandler()
	err := hook.Log(time.Time{}, 0, "", []interface{}{}, errToLog, log.Data{
		Context: s.ctx,
	})

	s.NoError(err)
	s.span.AssertExpectations(s.T())
}

func TestLoggingSuite(t *testing.T) {
	suite.Run(t, new(LoggingSuite))
}
