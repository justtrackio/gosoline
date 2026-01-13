package smplctx_test

import (
	"context"
	"testing"

	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
	"github.com/stretchr/testify/suite"
)

type ContextTestSuite struct {
	suite.Suite
}

func TestContextTestSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}

func (s *ContextTestSuite) TestGetSampling() {
	// Test when context is nil
	//nolint:staticcheck // testing behavior on nil context
	sampling := smplctx.GetSampling(nil)
	s.True(sampling.Sampled, "GetSampling(nil) should default to Sampled=true")

	// Test when context is empty
	sampling = smplctx.GetSampling(context.Background())
	s.True(sampling.Sampled, "GetSampling(emptyCtx) should default to Sampled=true")

	// Test when context has sampling (Sampled=false)
	ctx := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: false})
	sampling = smplctx.GetSampling(ctx)
	s.False(sampling.Sampled, "GetSampling(ctxWithSampledFalse) should return Sampled=false")
}

func (s *ContextTestSuite) TestIsSampled() {
	// Test default behavior (nil context)
	//nolint:staticcheck // testing behavior on nil context
	s.True(smplctx.IsSampled(nil), "IsSampled(nil) should return true by default")

	// Test default behavior (empty context)
	s.True(smplctx.IsSampled(context.Background()), "IsSampled(emptyCtx) should return true by default")

	// Test explicitly sampled=true
	ctxTrue := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: true})
	s.True(smplctx.IsSampled(ctxTrue), "IsSampled(ctxTrue) should return true")

	// Test explicitly sampled=false
	ctxFalse := smplctx.WithSampling(context.Background(), smplctx.Sampling{Sampled: false})
	s.False(smplctx.IsSampled(ctxFalse), "IsSampled(ctxFalse) should return false")
}
