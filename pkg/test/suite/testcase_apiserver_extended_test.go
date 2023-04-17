package suite_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/suite"
	"github.com/stretchr/testify/assert"
)

type GatewayTestSuite struct {
	suite.Suite
	totalTests int32
}

func TestGatewayTestSuite(t *testing.T) {
	var s GatewayTestSuite
	suite.Run(t, &s)
	assert.Equal(t, int32(5), s.totalTests)
}

func (s *GatewayTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithSharedEnvironment(),
	}
}

func (s *GatewayTestSuite) SetupApiDefinitions() apiserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		return &apiserver.Definitions{}, nil
	}
}

func (s *GatewayTestSuite) TestSingleTest() *suite.ApiServerTestCase {
	return s.createTestCase()
}

func (s *GatewayTestSuite) TestSkipped() *suite.ApiServerTestCase {
	return nil
}

func (s *GatewayTestSuite) TestMultipleTests() []*suite.ApiServerTestCase {
	return []*suite.ApiServerTestCase{
		s.createTestCase(),
		s.createTestCase(),
	}
}

func (s *GatewayTestSuite) TestMultipleTestsWithNil() []*suite.ApiServerTestCase {
	return []*suite.ApiServerTestCase{
		s.createTestCase(),
		nil,
		s.createTestCase(),
	}
}

func (s *GatewayTestSuite) createTestCase() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method:             http.MethodGet,
		Url:                "/health",
		Headers:            map[string]string{},
		Body:               struct{}{},
		ExpectedStatusCode: http.StatusOK,
		Assert: func(response *resty.Response) error {
			// language=JSON
			expectedResponse := `{}`
			s.JSONEq(expectedResponse, string(response.Body()))
			atomic.AddInt32(&s.totalTests, 1)

			return nil
		},
	}
}
