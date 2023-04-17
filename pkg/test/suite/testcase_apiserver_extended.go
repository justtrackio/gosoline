package suite

import (
	"net/http"
	"reflect"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
)

func init() {
	testCaseDefinitions["apiServerExtended"] = testCaseDefinition{
		matcher: isTestCaseApiServerExtended,
		builder: buildTestCaseApiServerExtended,
	}
}

type ApiServerTestCase struct {
	Method  string
	Url     string
	Headers map[string]string
	Body    interface{}
	// ExpectedStatusCode describes the status code the last response is required to have.
	ExpectedStatusCode int
	// ExpectedRedirectsToFollow describes the number of redirects we want to follow. It is an error if less redirects
	// are performed. More redirects result in the last redirect being returned as the response instead (e.g., if it is
	// to some external site or with a protocol different from HTTP(S) like intent://) and do not result in an error.
	ExpectedRedirectsToFollow int
	// ExpectedResult defines the *type* the final response should be parsed into. You can then access the unmarshalled
	// response in response.Result().
	ExpectedResult interface{}
	// ExpectedErr is compared with the error returned by the HTTP request. Only the error messages have to match.
	ExpectedErr error
	// Assert allows you to provide an assertion function that can be passed to validate certain post conditions (like
	// messages being written to the correct queues).
	// It also allows to check that the response carries the correct response body, redirects to the correct
	//	// location, or has the correct headers set. You don't need to validate the response status here, this is already
	//	// performed automatically.
	Assert func(response *resty.Response) error
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

func isTestCaseApiServerExtended(method reflect.Method) bool {
	if method.Func.Type().NumIn() != 1 {
		return false
	}

	if method.Func.Type().NumOut() != 1 {
		return false
	}

	actualType0 := method.Func.Type().Out(0)
	expectedType := reflect.TypeOf((*ApiServerTestCase)(nil))
	expectedSliceType := reflect.TypeOf([]*ApiServerTestCase{})

	return actualType0 == expectedType || actualType0 == expectedSliceType
}

func buildTestCaseApiServerExtended(suite TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return runTestCaseApiServer(suite, func(suite TestingSuite, app *appUnderTest, client *resty.Client) {
		out := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})[0].Interface()

		var err error
		var testCases []*ApiServerTestCase

		if tc, ok := out.(*ApiServerTestCase); ok {
			testCases = []*ApiServerTestCase{tc}
		} else {
			testCases = out.([]*ApiServerTestCase)
		}

		testCases = funk.Filter(testCases, func(elem *ApiServerTestCase) bool {
			return elem != nil
		})

		responses := make([]*resty.Response, len(testCases))
		remainingRedirects := 0

		client.SetRedirectPolicy(
			resty.RedirectPolicyFunc(func(request *http.Request, _ []*http.Request) error {
				if remainingRedirects <= 0 {
					return http.ErrUseLastResponse
				}

				remainingRedirects--

				return nil
			}),
		)

		for i, tc := range testCases {
			remainingRedirects = tc.ExpectedRedirectsToFollow
			responses[i], err = tc.request(client)

			assert.NotNil(suite.T(), responses[i], "there should be a response returned")

			if responses[i] != nil {
				assert.Equal(suite.T(), tc.ExpectedStatusCode, responses[i].StatusCode(), "response status code should match")
				assert.Equalf(suite.T(), 0, remainingRedirects, "expected %d redirects, but only %d redirects where performed", tc.ExpectedRedirectsToFollow, tc.ExpectedRedirectsToFollow-remainingRedirects)
			}

			if tc.ExpectedErr == nil {
				assert.NoError(suite.T(), err)
			} else {
				assert.EqualError(suite.T(), err, tc.ExpectedErr.Error())
			}
		}

		app.Stop()
		app.WaitDone()

		for i, tc := range testCases {
			if tc.Assert != nil {
				if err := tc.Assert(responses[i]); err != nil {
					assert.FailNow(suite.T(), err.Error(), "there should be no error on assert")
				}
			}
		}
	})
}
