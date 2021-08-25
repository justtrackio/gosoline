package suite

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
)

func init() {
	testCaseDefinitions["apiServer"] = testCaseDefinition{
		matcher: isTestCaseApiServer,
		builder: buildTestCaseApiServer,
	}
}

type TestingSuiteApiDefinitionsAware interface {
	SetupApiDefinitions() apiserver.Definer
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
	// Assert allows you to provide an assertion function which can be passed to validate certain post conditions (like
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

func isTestCaseApiServer(method reflect.Method) bool {
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

func buildTestCaseApiServer(suite TestingSuite, method reflect.Method) (testCaseRunner, error) {
	var ok bool
	var apiDefinitionAware TestingSuiteApiDefinitionsAware
	var server *apiserver.ApiServer

	if apiDefinitionAware, ok = suite.(TestingSuiteApiDefinitionsAware); !ok {
		return nil, fmt.Errorf("the suite has to implement the TestingSuiteApiDefinitionsAware interface to be able to run apiserver test cases")
	}

	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		// we first have to setup t, otherwise the test suite can't assert that there are no errors when setting up
		// route definitions or test cases
		suite.SetT(t)

		out := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})[0].Interface()

		var testCases []*ApiServerTestCase
		if tc, ok := out.(*ApiServerTestCase); ok {
			testCases = []*ApiServerTestCase{tc}
		} else {
			testCases = out.([]*ApiServerTestCase)
		}

		responses := make([]*resty.Response, len(testCases))
		apiDefinitions := apiDefinitionAware.SetupApiDefinitions()

		suiteOptions.appModules["api"] = func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			module, err := apiserver.New(apiDefinitions)(ctx, config, logger)
			if err != nil {
				return nil, err
			}

			server = module.(*apiserver.ApiServer)

			return server, err
		}

		suiteOptions.addAppOption(application.WithConfigMap(map[string]interface{}{
			"api": map[string]interface{}{
				"port": 0,
			},
		}))

		runTestCaseApplication(t, suite, suiteOptions, environment, func(app *appUnderTest) {
			port, err := server.GetPort()
			if err != nil {
				assert.FailNow(t, err.Error(), "can not get port of server")
				return
			}

			remainingRedirects := 0
			url := fmt.Sprintf("http://127.0.0.1:%d", *port)

			client := resty.New().SetHostURL(url).SetRedirectPolicy(
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

				assert.NotNil(t, responses[i], "there should be a response returned")

				if responses[i] != nil {
					assert.Equal(t, tc.ExpectedStatusCode, responses[i].StatusCode(), "response status code should match")
					assert.Equalf(t, 0, remainingRedirects, "expected %d redirects, but only %d redirects where performed", tc.ExpectedRedirectsToFollow, tc.ExpectedRedirectsToFollow-remainingRedirects)
				}

				if tc.ExpectedErr == nil {
					assert.NoError(t, err)
				} else {
					assert.EqualError(t, err, tc.ExpectedErr.Error())
				}
			}

			app.Stop()
			app.WaitDone()

			for i, tc := range testCases {
				if tc.Assert != nil {
					if err := tc.Assert(responses[i]); err != nil {
						assert.FailNow(t, err.Error(), "there should be no error on assert")
					}
				}
			}
		})
	}, nil
}
