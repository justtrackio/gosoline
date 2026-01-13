package smpl_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/suite"
)

type StrategiesTestSuite struct {
	suite.Suite
}

func TestStrategiesTestSuite(t *testing.T) {
	suite.Run(t, new(StrategiesTestSuite))
}

func (s *StrategiesTestSuite) TestDecideByAlways() {
	applied, sampled, err := smpl.DecideByAlways(context.Background())
	s.True(applied, "DecideByAlways should always be applied")
	s.True(sampled, "DecideByAlways should always return sampled=true")
	s.NoError(err, "DecideByAlways should return no error")
}

func (s *StrategiesTestSuite) TestDecideByNever() {
	applied, sampled, err := smpl.DecideByNever(context.Background())
	s.True(applied, "DecideByNever should always be applied")
	s.False(sampled, "DecideByNever should always return sampled=false")
	s.NoError(err, "DecideByNever should return no error")
}

func (s *StrategiesTestSuite) TestDecideByTracing() {
	// 1. No trace in context
	applied, _, err := smpl.DecideByTracing(context.Background())
	s.False(applied, "DecideByTracing should not apply if no trace is present")
	s.NoError(err)

	// 2. Trace in context with Sampled=true
	ctxTrue := tracing.ContextWithTrace(context.Background(), &tracing.Trace{Sampled: true})
	applied, sampled, err := smpl.DecideByTracing(ctxTrue)
	s.True(applied, "DecideByTracing should apply if trace is present")
	s.True(sampled, "DecideByTracing should return sampled=true if trace.Sampled=true")
	s.NoError(err)

	// 3. Trace in context with Sampled=false
	ctxFalse := tracing.ContextWithTrace(context.Background(), &tracing.Trace{Sampled: false})
	applied, sampled, err = smpl.DecideByTracing(ctxFalse)
	s.True(applied, "DecideByTracing should apply if trace is present")
	s.False(sampled, "DecideByTracing should return sampled=false if trace.Sampled=false")
	s.NoError(err)
}
