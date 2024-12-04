package suite

import (
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	testCaseDefinitions["stream"] = testCaseDefinition{
		matcher: isTestCaseStream,
		builder: buildTestCaseStream,
	}
}

type StreamTestCaseInput struct {
	Attributes map[string]string
	Body       interface{}
}

type StreamTestCaseOutput struct {
	Model              interface{}
	ExpectedAttributes map[string]string
	ExpectedBody       interface{}
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

func isTestCaseStream(method reflect.Method) bool {
	if method.Func.Type().NumIn() != 1 {
		return false
	}

	if method.Func.Type().NumOut() != 1 {
		return false
	}

	actualType0 := method.Func.Type().Out(0)
	expectedType := reflect.TypeOf((*StreamTestCase)(nil))
	expectedProviderType := reflect.TypeOf((*ToStreamTestCase)(nil)).Elem()

	return actualType0 == expectedType || actualType0 == expectedProviderType || isTestCaseMapStream(method)
}

func isTestCaseMapStream(method reflect.Method) bool {
	actualType0 := method.Func.Type().Out(0)
	expectedMapType := reflect.TypeOf(map[string]*StreamTestCase{})
	expectedProviderMapType := reflect.TypeOf(map[string]ToStreamTestCase{})

	return actualType0 == expectedMapType ||
		actualType0 == expectedProviderMapType
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
		name := name
		testCasesProvider := testCasesProvider
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
