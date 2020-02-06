package tracing_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/applike/gosoline/pkg/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type LoggingSuite struct {
	suite.Suite

	ctx  context.Context
	span *mocks.Span
}

func TestLoggingSuite(t *testing.T) {
	suite.Run(t, new(LoggingSuite))
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

	assert.Contains(s.T(), fields, "trace_id")
	assert.Equal(s.T(), "1-5e3d5273-7f0bd984ad68e2d290caeb84", fields["trace_id"])
	s.span.AssertExpectations(s.T())
}

func (s *LoggingSuite) TestLoggerErrorHook() {
	errToLog := fmt.Errorf("unexpected error")
	s.span.On("AddError", errToLog)

	hook := tracing.NewLoggerErrorHook()
	err := hook.Fire("", "", errToLog, &mon.Metadata{
		Context: s.ctx,
	})

	assert.NoError(s.T(), err)
	s.span.AssertExpectations(s.T())
}
