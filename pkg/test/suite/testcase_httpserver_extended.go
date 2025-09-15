package suite

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterTestCaseDefinition("httpserverExtended", isTestCaseHttpserverExtended, buildTestCaseHttpserverExtended)
}

const expectedTestCaseHttpserverExtendedSignature = "func (s TestingSuite) TestFunc() T"

// A HttpserverTestCase should be the return value from your test functions. The following signatures are supported:
//
// func (s* Suite) TestXXX() *HttpserverTestCase
// func (s* Suite) TestXXX() []*HttpserverTestCase
// func (s* Suite) TestXXX() ToHttpserverTestCaseList
// func (s* Suite) TestXXX() map[string]*HttpserverTestCase
// func (s* Suite) TestXXX() map[string][]*HttpserverTestCase
// func (s* Suite) TestXXX() map[string]ToHttpserverTestCaseList
type HttpserverTestCase struct {
	Method  string
	Url     string
	Headers map[string]string
	// Body will be used as the request body. Supported request body data types is `string`,
	// `[]byte`, `struct`, `map`, `slice` and `io.Reader`. Body value can be pointer or non-pointer.
	// Automatic marshalling for JSON and XML content type, if it is `struct`, `map`, or `slice`.
	//
	// If you call EncodeBodyProtobuf on your body before assigning it to this field, it will instead be
	// encoded using protobuf.
	//
	// To send the contents of a file, you can use ReadBodyFile and assign the result to this field. The
	// test suite will read the file contents and send it as your request.
	Body any
	// ExpectedStatusCode describes the status code the last response is required to have.
	ExpectedStatusCode int
	// ExpectedRedirectsToFollow describes the number of redirects we want to follow. It is an error if less redirects
	// are performed. More redirects result in the last redirect being returned as the response instead (e.g., if it is
	// to some external site or with a protocol different from HTTP(S) like intent://) and do not result in an error.
	ExpectedRedirectsToFollow int
	// ExpectedResult defines the *type* the final response should be parsed into. You can then access the unmarshalled
	// response in response.Result().
	ExpectedResult any
	// ExpectedErr is compared with the error returned by the HTTP request. Only the error messages have to match.
	ExpectedErr error
	// Assert allows you to provide an assertion function that can be passed to validate certain post conditions (like
	// messages being written to the correct queues).
	// It also allows to check that the response carries the correct response body, redirects to the correct
	//	// location, or has the correct headers set. You don't need to validate the response status here, this is already
	//	// performed automatically.
	Assert func(response *resty.Response) error
	// AssertResultFile can be used to read the expected response body from a file, which will be used to check equality
	// with the actual response body. If the file name extension is .json, the equality check is done via assert.JSONEq.
	AssertResultFile string
}

// A ToHttpserverTestCaseList can be converted into a list of test cases. You can use this interface instead of concrete
// test case types when declaring test cases.
type ToHttpserverTestCaseList interface {
	ToTestCaseList() []*HttpserverTestCase
}

// A HttpserverTestCaseListProvider allows you to provide a function creating test cases instead of test cases directly.
// This is useful as it runs after the app has been constructed, fixtures loaded, etc.
type HttpserverTestCaseListProvider func() []*HttpserverTestCase

type ProtobufEncodable interface {
	ToMessage() (proto.Message, error)
}

type encodeBodyProtobuf struct {
	ProtobufEncodable
}

func EncodeBodyProtobuf(body ProtobufEncodable) any {
	return encodeBodyProtobuf{
		ProtobufEncodable: body,
	}
}

type readBodyFile struct {
	file string
}

func ReadBodyFile(bodyFile string) any {
	return readBodyFile{
		file: bodyFile,
	}
}

func (f HttpserverTestCaseListProvider) ToTestCaseList() []*HttpserverTestCase {
	return f()
}

func (c *HttpserverTestCase) ToTestCaseList() []*HttpserverTestCase {
	if c == nil {
		return nil
	}

	return []*HttpserverTestCase{
		c,
	}
}

func (c *HttpserverTestCase) request(client *resty.Client) (*resty.Response, error) {
	req := client.R()

	if c.Headers != nil {
		req.SetHeaders(c.Headers)
	}

	if c.Body != nil {
		switch body := c.Body.(type) {
		case encodeBodyProtobuf:
			msg, err := body.ToMessage()
			if err != nil {
				return nil, fmt.Errorf("failed to convert body to message: %w", err)
			}

			bytes, err := proto.Marshal(msg)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal message as protobuf: %w", err)
			}

			req.SetBody(bytes)
		case readBodyFile:
			bytes, err := os.ReadFile(body.file)
			if err != nil {
				return nil, fmt.Errorf("can not read body from file %q: %w", body.file, err)
			}

			req.SetBody(bytes)
		default:
			req.SetBody(c.Body)
		}
	}

	if c.ExpectedResult != nil {
		req.SetResult(c.ExpectedResult)
	}

	return req.Execute(c.Method, c.Url)
}

func isTestCaseHttpserverExtended(s TestingSuite, method reflect.Method) error {
	if _, ok := s.(TestingSuiteApiDefinitionsAware); !ok {
		return fmt.Errorf("the suite has to implement the TestingSuiteApiDefinitionsAware interface to be able to run httpserver test cases")
	}

	if method.Func.Type().NumIn() != 1 {
		return fmt.Errorf("expected %q, but function has %d arguments", expectedTestCaseHttpserverExtendedSignature, method.Func.Type().NumIn())
	}

	if method.Func.Type().NumOut() != 1 {
		return fmt.Errorf("expected %q, but function has %d return values", expectedTestCaseHttpserverExtendedSignature, method.Func.Type().NumOut())
	}

	actualType0 := method.Func.Type().In(0)
	expectedType0 := reflect.TypeOf((*TestingSuite)(nil)).Elem()

	if !actualType0.Implements(expectedType0) {
		return fmt.Errorf("expected %q, but first argument type/receiver type is %s", expectedTestCaseHttpserverExtendedSignature, actualType0.String())
	}

	actualTypeResult := method.Func.Type().Out(0)
	expectedType := reflect.TypeOf((*HttpserverTestCase)(nil))
	expectedSliceType := reflect.TypeOf([]*HttpserverTestCase{})
	expectedProviderType := reflect.TypeOf((*ToHttpserverTestCaseList)(nil)).Elem()

	if actualTypeResult != expectedType &&
		actualTypeResult != expectedSliceType &&
		actualTypeResult != expectedProviderType &&
		!isTestCaseMapHttpserverExtended(method) {
		return fmt.Errorf(
			"expected %q, but return type is %s. Allowed return types are:\n - %s",
			expectedTestCaseHttpserverExtendedSignature,
			actualTypeResult.String(),
			strings.Join([]string{
				"*HttpserverTestCase",
				"[]*HttpserverTestCase",
				"ToHttpserverTestCaseList",
				"map[string]*HttpserverTestCase",
				"map[string][]*HttpserverTestCase",
				"map[string]ToHttpserverTestCaseList",
			}, "\n - "),
		)
	}

	return nil
}

func isTestCaseMapHttpserverExtended(method reflect.Method) bool {
	actualTypeResult := method.Func.Type().Out(0)
	expectedMapType := reflect.TypeOf(map[string]*HttpserverTestCase{})
	expectedMapSliceType := reflect.TypeOf(map[string][]*HttpserverTestCase{})
	expectedProviderMapType := reflect.TypeOf(map[string]ToHttpserverTestCaseList{})

	return actualTypeResult == expectedMapType ||
		actualTypeResult == expectedMapSliceType ||
		actualTypeResult == expectedProviderMapType
}

func buildTestCaseHttpserverExtended(suite TestingSuite, method reflect.Method) (TestCaseRunner, error) {
	if isTestCaseMapHttpserverExtended(method) {
		out := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})[0].Interface()

		var providerMap map[string]ToHttpserverTestCaseList

		switch testCases := out.(type) {
		case map[string]*HttpserverTestCase:
			providerMap = funk.MapValues(testCases, func(value *HttpserverTestCase) ToHttpserverTestCaseList {
				return value
			})
		case map[string][]*HttpserverTestCase:
			providerMap = funk.MapValues(testCases, func(value []*HttpserverTestCase) ToHttpserverTestCaseList {
				return HttpserverTestCaseListProvider(func() []*HttpserverTestCase {
					return value
				})
			})
		case map[string]ToHttpserverTestCaseList:
			providerMap = testCases
		}

		return runHttpServerExtendedTestsMap(suite, providerMap)
	}

	return runHttpServerExtendedTests(suite, func() []*HttpserverTestCase {
		out := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})[0].Interface()

		var testCases []*HttpserverTestCase

		switch tc := out.(type) {
		case *HttpserverTestCase:
			testCases = tc.ToTestCaseList()
		case ToHttpserverTestCaseList:
			if tc != nil {
				testCases = tc.ToTestCaseList()
			}
		case []*HttpserverTestCase:
			testCases = tc
		}

		return testCases
	})
}

func runHttpServerExtendedTestsMap(suite TestingSuite, testCases map[string]ToHttpserverTestCaseList) (TestCaseRunner, error) {
	testCaseRunners := make([]TestCaseRunner, 0, len(testCases))

	for name, testCasesProvider := range testCases {
		runner, err := runHttpServerExtendedTests(suite, func() []*HttpserverTestCase {
			if testCasesProvider == nil {
				return nil
			}

			return testCasesProvider.ToTestCaseList()
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create http test case runner for %q: %w", name, err)
		}

		testCaseRunners = append(testCaseRunners, func(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, environment *env.Environment) {
			t.Run(name, func(t *testing.T) {
				runner(t, suite, suiteConf, environment)
			})
		})
	}

	return func(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, environment *env.Environment) {
		if len(testCaseRunners) == 0 {
			suite.T().SkipNow()
		}

		for _, testCaseRunner := range testCaseRunners {
			testCaseRunner(t, suite, suiteConf, environment)
		}
	}, nil
}

func runHttpServerExtendedTests(suite TestingSuite, getTestCases func() []*HttpserverTestCase) (TestCaseRunner, error) {
	return runTestCaseHttpserver(suite, func(suite TestingSuite, app AppUnderTest, client *resty.Client) {
		RunTestCaseInSuite(suite.T(), suite, func() {
			testCases := funk.Filter(getTestCases(), func(elem *HttpserverTestCase) bool {
				return elem != nil
			})

			if len(testCases) == 0 {
				app.Stop()
				app.WaitDone()
				suite.T().SkipNow()
			}

			responses := runHttpServerExtendedRequests(suite, testCases, client)

			app.Stop()
			app.WaitDone()

			verifyHttpServerExtendedResponses(suite, testCases, responses)
		})
	})
}

func runHttpServerExtendedRequests(suite TestingSuite, testCases []*HttpserverTestCase, client *resty.Client) []*resty.Response {
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
		var err error

		remainingRedirects = tc.ExpectedRedirectsToFollow
		responses[i], err = tc.request(client)

		assert.NotNil(suite.T(), responses[i], "there should be a response returned")

		if responses[i] != nil {
			assert.Equal(suite.T(), tc.ExpectedStatusCode, responses[i].StatusCode(), "response status code should match")
			assert.Equalf(
				suite.T(),
				0,
				remainingRedirects,
				"expected %d redirects, but only %d redirects where performed",
				tc.ExpectedRedirectsToFollow,
				tc.ExpectedRedirectsToFollow-remainingRedirects,
			)
		}

		if tc.ExpectedErr == nil {
			assert.NoError(suite.T(), err)
		} else {
			assert.EqualError(suite.T(), err, tc.ExpectedErr.Error())
		}
	}

	return responses
}

func verifyHttpServerExtendedResponses(suite TestingSuite, testCases []*HttpserverTestCase, responses []*resty.Response) {
	for i, tc := range testCases {
		if tc.Assert != nil {
			if err := tc.Assert(responses[i]); err != nil {
				assert.FailNow(suite.T(), err.Error(), "there should be no error on assert")
			}
		}

		if tc.AssertResultFile != "" {
			var bytes []byte
			var err error

			if bytes, err = os.ReadFile(tc.AssertResultFile); err != nil {
				assert.FailNow(suite.T(), err.Error(), "can not read result file %q", tc.AssertResultFile)
			}

			extension := path.Ext(tc.AssertResultFile)
			actual := responses[i].Body()

			switch extension {
			case ".json":
				assert.JSONEq(suite.T(), string(bytes), string(actual), "response doesn't match")
			default:
				assert.Equal(suite.T(), bytes, actual, "response doesn't match")
			}
		}
	}
}
