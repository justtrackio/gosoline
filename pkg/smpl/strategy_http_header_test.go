package smpl_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/stretchr/testify/suite"
)

type HttpHeaderStrategyTestSuite struct {
	suite.Suite
}

func TestHttpHeaderStrategyTestSuite(t *testing.T) {
	suite.Run(t, new(HttpHeaderStrategyTestSuite))
}

func (s *HttpHeaderStrategyTestSuite) TestDecideByHttpHeader_Missing() {
	req := &http.Request{Header: make(http.Header)}
	strategy := smpl.DecideByHttpHeader(req)

	applied, _, err := strategy(context.Background())
	s.False(applied, "Strategy should NOT apply if header is missing")
	s.NoError(err)
}

func (s *HttpHeaderStrategyTestSuite) TestDecideByHttpHeader_True() {
	req := &http.Request{Header: make(http.Header)}
	req.Header.Set(smpl.HeaderSamplingKey, "true")
	strategy := smpl.DecideByHttpHeader(req)

	applied, sampled, err := strategy(context.Background())
	s.True(applied, "Strategy SHOULD apply if header is present")
	s.True(sampled, "Should be sampled=true for header value 'true'")
	s.NoError(err)
}

func (s *HttpHeaderStrategyTestSuite) TestDecideByHttpHeader_False() {
	req := &http.Request{Header: make(http.Header)}
	req.Header.Set(smpl.HeaderSamplingKey, "false")
	strategy := smpl.DecideByHttpHeader(req)

	applied, sampled, err := strategy(context.Background())
	s.True(applied, "Strategy SHOULD apply if header is present")
	s.False(sampled, "Should be sampled=false for header value 'false'")
	s.NoError(err)
}

func (s *HttpHeaderStrategyTestSuite) TestDecideByHttpHeader_Numeric() {
	req := &http.Request{Header: make(http.Header)}
	req.Header.Set(smpl.HeaderSamplingKey, "1")
	strategy := smpl.DecideByHttpHeader(req)

	applied, sampled, err := strategy(context.Background())
	s.True(applied, "Strategy SHOULD apply")
	s.True(sampled, "Should support numeric booleans like '1'")
	s.NoError(err)
}

func (s *HttpHeaderStrategyTestSuite) TestDecideByHttpHeader_Invalid() {
	req := &http.Request{Header: make(http.Header)}
	req.Header.Set(smpl.HeaderSamplingKey, "not-a-bool")
	strategy := smpl.DecideByHttpHeader(req)

	applied, _, err := strategy(context.Background())
	s.False(applied, "Strategy application status on error is implementation detail, but typically returns false/false/err")
	s.Error(err)
	s.Contains(err.Error(), "could not parse sampling header value 'not-a-bool'")
}
