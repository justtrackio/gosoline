package suite

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/test/env"
)

func init() {
	registerTestCaseDefinition("base", isTestCaseBase, buildTestCaseBase)
}

const expectedTestCaseBaseSignature = "func (s TestingSuite) TestFunc()"

func isTestCaseBase(method reflect.Method) error {
	if method.Func.Type().NumIn() != 1 {
		return fmt.Errorf("expected %q, but function has %d arguments", expectedTestCaseBaseSignature, method.Func.Type().NumIn())
	}

	if method.Func.Type().NumOut() != 0 {
		return fmt.Errorf("expected %q, but function has %d return values", expectedTestCaseBaseSignature, method.Func.Type().NumOut())
	}

	actualType0 := method.Func.Type().In(0)
	expectedType0 := reflect.TypeOf((*TestingSuite)(nil)).Elem()

	if !actualType0.Implements(expectedType0) {
		return fmt.Errorf("expected %q, but first argument type/receiver type is %s", expectedTestCaseBaseSignature, actualType0.String())
	}

	return nil
}

func buildTestCaseBase(_ TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		suite.SetT(t)
		method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
	}, nil
}
