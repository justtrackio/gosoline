package suite

import (
	"github.com/applike/gosoline/pkg/test/env"
	"reflect"
	"testing"
)

func init() {
	testCaseDefinitions["base"] = testCaseDefinition{
		matcher: isTestCaseBase,
		builder: buildTestCaseBase,
	}
}

func isTestCaseBase(method reflect.Method) bool {
	if method.Func.Type().NumIn() != 1 {
		return false
	}

	return method.Func.Type().NumOut() == 0
}

func buildTestCaseBase(suite TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		suite.SetT(t)
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}, nil
}
