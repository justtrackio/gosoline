package suite

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	registerTestCaseDefinition("stream", isTestCaseStream, buildTestCaseStream)
}

const expectedTestCaseStreamSignature = "func (s TestingSuite) TestFunc() T"

type StreamTestCaseInput struct {
	Attributes map[string]string
	Body       any
}

type StreamTestCaseOutput struct {
	Model              any
	ExpectedAttributes map[string]string
	ExpectedBody       any
}

type StreamTestCase struct {
	Input  map[string][]StreamTestCaseInput
	Output map[string][]StreamTestCaseOutput
	Assert func() error
}

type ToStreamTestCase interface {
	ToTestCase() *StreamTestCase
}

type StreamTestCaseProvider func() *StreamTestCase

func (f StreamTestCaseProvider) ToTestCase() *StreamTestCase {
	return f()
}

func (c *StreamTestCase) ToTestCase() *StreamTestCase {
	return c
}

func isTestCaseStream(method reflect.Method) error {
	if method.Func.Type().NumIn() != 1 {
		return fmt.Errorf("expected %q, but function has %d arguments", expectedTestCaseStreamSignature, method.Func.Type().NumIn())
	}

	if method.Func.Type().NumOut() != 1 {
		return fmt.Errorf("expected %q, but function has %d return values", expectedTestCaseStreamSignature, method.Func.Type().NumOut())
	}

	actualType0 := method.Func.Type().In(0)
	expectedType0 := reflect.TypeOf((*TestingSuite)(nil)).Elem()

	if !actualType0.Implements(expectedType0) {
		return fmt.Errorf("expected %q, but first argument type/receiver type is %s", expectedTestCaseStreamSignature, actualType0.String())
	}

	actualTypeResult := method.Func.Type().Out(0)
	expectedType := reflect.TypeOf((*StreamTestCase)(nil))
	expectedProviderType := reflect.TypeOf((*ToStreamTestCase)(nil)).Elem()

	if actualTypeResult != expectedType &&
		actualTypeResult != expectedProviderType &&
		!isTestCaseMapStream(method) {
		return fmt.Errorf(
			"expected %q, but return type is %s. Allowed return types are:\n - %s",
			expectedTestCaseStreamSignature,
			actualTypeResult.String(),
			strings.Join([]string{
				"*StreamTestCase",
				"ToStreamTestCase",
				"map[string]*StreamTestCase",
				"map[string]ToStreamTestCase",
			}, "\n - "),
		)
	}

	return nil
}

func isTestCaseMapStream(method reflect.Method) bool {
	actualTypeResult := method.Func.Type().Out(0)
	expectedMapType := reflect.TypeOf(map[string]*StreamTestCase{})
	expectedProviderMapType := reflect.TypeOf(map[string]ToStreamTestCase{})

	return actualTypeResult == expectedMapType ||
		actualTypeResult == expectedProviderMapType
}

func buildTestCaseStream(suite TestingSuite, method reflect.Method) (testCaseRunner, error) {
	if isTestCaseMapStream(method) {
		out := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})[0].Interface()

		var providerMap map[string]ToStreamTestCase

		switch testCases := out.(type) {
		case map[string]*StreamTestCase:
			providerMap = funk.MapValues(testCases, func(value *StreamTestCase) ToStreamTestCase {
				return value
			})
		case map[string]ToStreamTestCase:
			providerMap = testCases
		}

		return runStreamTestMap(providerMap)
	}

	return runStreamTest(func() *StreamTestCase {
		out := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})[0].Interface()

		switch tc := out.(type) {
		case *StreamTestCase:
			return tc
		case ToStreamTestCase:
			if tc != nil {
				return tc.ToTestCase()
			}
		}

		return nil
	}), nil
}

func runStreamTestMap(testCases map[string]ToStreamTestCase) (testCaseRunner, error) {
	testCaseRunners := make([]testCaseRunner, 0, len(testCases))

	for name, testCasesProvider := range testCases {
		runner := runStreamTest(func() *StreamTestCase {
			if testCasesProvider == nil {
				return nil
			}

			return testCasesProvider.ToTestCase()
		})

		testCaseRunners = append(testCaseRunners, func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
			t.Run(name, func(t *testing.T) {
				runner(t, suite, suiteOptions, environment)
			})
		})
	}

	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		if len(testCaseRunners) == 0 {
			t.SkipNow()
		}

		for _, testCaseRunner := range testCaseRunners {
			testCaseRunner(t, suite, suiteOptions, environment)
		}
	}, nil
}

func runStreamTest(getTestCase func() *StreamTestCase) testCaseRunner {
	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		runTestCaseApplication(t, suite, suiteOptions, environment, func(app *appUnderTest) {
			runTestCaseInSuite(t, suite, func() {
				tc := getTestCase()

				if tc == nil {
					app.Stop()
					app.WaitDone()
					t.SkipNow()
				}

				writeStreamTestInputData(tc, suite)
				app.WaitDone()
				assertStreamTestOutputs(tc, suite, t)
			})
		})
	}
}

func writeStreamTestInputData(tc *StreamTestCase, suite TestingSuite) {
	for inputName, data := range tc.Input {
		input := suite.Env().StreamInput(inputName)

		for _, d := range data {
			input.Publish(d.Body, d.Attributes)
		}

		input.Stop()
	}
}

func assertStreamTestOutputs(tc *StreamTestCase, suite TestingSuite, t *testing.T) {
	for outputName, data := range tc.Output {
		output := suite.Env().StreamOutput(outputName)

		for i, d := range data {
			model := d.Model
			attrs := output.Unmarshal(i, model)

			assert.Equal(t, d.ExpectedAttributes, attrs, "attributes do not match")
			assert.Equal(t, d.ExpectedBody, model, "body does not match")
		}
	}

	if tc.Assert != nil {
		assert.NoError(t, tc.Assert(), "there should be no error happening on assert")
	}
}
