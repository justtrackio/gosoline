//go:build integration && fixtures

// snippet-start: imports
package apitest

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/test/suite"
)

// snippet-end: imports

// snippet-start: test suite
type HttpTestSuite struct {
	suite.Suite

	clock clock.Clock
}

// snippet-end: test suite

// snippet-start: set up suite
func (s *HttpTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		// A hard-coded log level.
		suite.WithLogLevel("info"),

		// Configurations from a config file.
		suite.WithConfigFile("./config.dist.yml"),

		// The fixture set you created in the last section.
		suite.WithFixtureSetFactory(fixtureSetsFactory),

		// suite.WithClockProvider(s.clock),
		suite.WithClockProvider(s.clock),
	}
}

// snippet-end: set up suite

// snippet-start: set up api defs
func (s *HttpTestSuite) SetupApiDefinitions() httpserver.Definer {
	return ApiDefiner
}

// snippet-end: set up api defs

// snippet-start: test to euro
func (s *HttpTestSuite) Test_ToEuro(_ suite.AppUnderTest, client *resty.Client) error {
	var result float64

	// Make a GET request to /euro/:amount/:currency where :amount = 10 and :currency = GBP
	response, err := client.R().
		SetResult(&result).
		Execute(http.MethodGet, "/euro/10/GBP")

	// Check that there is no error.
	s.NoError(err)

	// Check that the response status code is 200 OK.
	s.Equal(http.StatusOK, response.StatusCode())

	// Check that the converted amount is 8.0.
	s.Equal(8.0, result)

	return nil
}

// snippet-end: test to euro

// snippet-start: test-toeuroatdate
func (s *HttpTestSuite) Test_ToEuroAtDate(_ suite.AppUnderTest, client *resty.Client) error {
	var result float64

	response, err := client.R().
		SetResult(&result).
		Execute(http.MethodGet, "/euro-at-date/10/GBP/2021-01-03T00:00:00Z")

	s.NoError(err)
	s.Equal(http.StatusOK, response.StatusCode())
	s.Equal(12.5, result)

	return nil
}

// snippet-end: test-toeuroatdate

// snippet-start: test euro
func (s *HttpTestSuite) Test_Euro() *suite.HttpserverTestCase {
	return &suite.HttpserverTestCase{
		Method:             http.MethodGet,
		Url:                "/euro/10/GBP",
		Headers:            map[string]string{},
		ExpectedStatusCode: http.StatusOK,
		Assert: func(response *resty.Response) error {
			result, err := strconv.ParseFloat(string(response.Body()), 64)
			s.NoError(err)
			s.Equal(8.0, result)

			return nil
		},
	}
}

// snippet-end: test euro

// snippet-start: unit test
func TestHttpTestSuite(t *testing.T) {
	suite.Run(t, &HttpTestSuite{
		clock: clock.NewFakeClockAt(time.Now().UTC()),
	})
}

// snippet-end: unit test
