//go:build integration && fixtures
// +build integration,fixtures

package integration

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/examples/getting_started/api/definer"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

type ApiTestSuite struct {
	suite.Suite

	clock clock.Clock
}

func (s *ApiTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithConfigFile("../api/config.dist.yml"),
		suite.WithFixtures(fixtureSets),
		suite.WithClockProvider(s.clock),
	}
}

func (s *ApiTestSuite) SetupApiDefinitions() apiserver.Definer {
	return definer.ApiDefiner
}

func (s *ApiTestSuite) Test_ToEuro(_ suite.AppUnderTest, client *resty.Client) error {
	var result float64

	response, err := client.R().
		SetResult(&result).
		Execute(http.MethodGet, "/euro/10/GBP")

	s.NoError(err)
	s.Equal(http.StatusOK, response.StatusCode())
	s.Equal(8.0, result)

	return nil
}

func (s *ApiTestSuite) Test_ToEuroAtDate(_ suite.AppUnderTest, client *resty.Client) error {
	var result float64

	response, err := client.R().
		SetResult(&result).
		Execute(http.MethodGet, "/euro-at-date/10/GBP/2021-01-03T00:00:00Z")

	s.NoError(err)
	s.Equal(http.StatusOK, response.StatusCode())
	s.Equal(12.5, result)

	return nil
}

func TestApiTestSuite(t *testing.T) {
	suite.Run(t, &ApiTestSuite{
		clock: clock.NewFakeClockAt(time.Now().UTC()),
	})
}
