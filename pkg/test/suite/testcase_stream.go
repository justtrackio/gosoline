package suite

import (
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func init() {
	testCaseDefinitions["stream"] = testCaseDefinition{
		matcher: isTestCaseStream,
		builder: buildTestCaseStream,
	}
}

type StreamTestCaseInput struct {
	Attributes map[string]interface{}
	Body       interface{}
}

type StreamTestCaseOutput struct {
	Model              interface{}
	ExpectedAttributes map[string]interface{}
	ExpectedBody       interface{}
}

type StreamTestCase struct {
	Input  map[string][]StreamTestCaseInput
	Output map[string][]StreamTestCaseOutput
	Assert func() error
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

	return actualType0 == expectedType
}

func buildTestCaseStream(suite TestingSuite, method reflect.Method) (testCaseRunner, error) {
	out := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	tc := out[0].Interface().(*StreamTestCase)

	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		suite.SetT(t)

		runTestCaseApplication(t, suite, suiteOptions, environment, func(app *appUnderTest) {
			for inputName, data := range tc.Input {
				input := suite.Env().StreamInput(inputName)

				for _, d := range data {
					input.Publish(d.Body, d.Attributes)
				}

				input.Stop()
			}

			app.WaitDone()

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
				if err := tc.Assert(); err != nil {
					assert.NoError(t, err, "there should be no error happening on assert")
				}
			}
		})
	}, nil
}
