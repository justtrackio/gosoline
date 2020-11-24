package test

import (
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"gopkg.in/resty.v1"
	"testing"
)

type ApiServerTestCase struct {
	Method             string
	Url                string
	Headers            map[string]string
	Body               interface{}
	ExpectedStatusCode int
	ExpectedResult     interface{}
	ExpectedErr        error
	Assert             func() error
}

func (c ApiServerTestCase) request(client *resty.Client) (*resty.Response, error) {
	req := client.R()

	if c.Headers != nil {
		req.SetHeaders(c.Headers)
	}

	if c.Body != nil {
		req.SetBody(c.Body)
	}

	if c.ExpectedResult != nil {
		req.SetResult(c.ExpectedResult)
	}

	return req.Execute(c.Method, c.Url)
}

type TestingSuiteApiServer interface {
	TestingSuite
	SetupApiDefinitions() apiserver.Define
	SetupTestCases() []ApiServerTestCase
	TestApiServer(app AppUnderTest, server *apiserver.ApiServer, testCases []ApiServerTestCase)
}

func RunApiServerTestSuite(t *testing.T, suite TestingSuiteApiServer) {
	suite.SetT(t)

	server := apiserver.New(suite.SetupApiDefinitions())
	testcase := func(appUnderTest AppUnderTest) {
		testCases := suite.SetupTestCases()
		suite.TestApiServer(appUnderTest, server, testCases)
	}

	extraOptions := []SuiteOption{
		WithModule("api", server),
		WithConfigMap(map[string]interface{}{
			"api_port": 0,
		}),
	}

	RunTestCase(t, suite, testcase, extraOptions...)
}

type ApiServerTestSuite struct {
	Suite
}

func (s *ApiServerTestSuite) TestApiServer(app AppUnderTest, server *apiserver.ApiServer, testCases []ApiServerTestCase) {
	port, err := server.GetPort()

	if err != nil {
		s.FailNow(err.Error(), "can not get port of server")
		return
	}

	url := fmt.Sprintf("http://127.0.0.1:%d", *port)
	client := resty.New().SetHostURL(url)
	responses := make([]*resty.Response, len(testCases))

	for i, tc := range testCases {
		responses[i], err = tc.request(client)

		s.Equal(tc.ExpectedStatusCode, responses[i].StatusCode(), "response status code should match")

		if tc.ExpectedErr == nil {
			s.NoError(err)
		} else {
			s.EqualError(err, tc.ExpectedErr.Error())
		}
	}

	app.Stop()
	app.WaitDone()

	for _, tc := range testCases {
		if tc.Assert != nil {
			if err := tc.Assert(); err != nil {
				s.FailNowf(err.Error(), "there should be no error on assert")
			}
		}
	}
}
