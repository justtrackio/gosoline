package suite

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/env"
)

func init() {
	RegisterTestCaseDefinition("base", isTestCaseBase, buildTestCaseBase)
}

const expectedTestCaseBaseSignature = "func (s TestingSuite) TestFunc()"

func isTestCaseBase(_ TestingSuite, method reflect.Method) error {
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

func buildTestCaseBase(_ TestingSuite, method reflect.Method) (TestCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, environment *env.Environment) {
		suite.SetT(t)

		suiteConf.appModules["testcase"] = func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return kernel.NewModuleFunc(func(ctx context.Context) error {
				return nil
			}), nil
		}

		// Use the environment's context so that lifecycle managers registered during SetupTest
		// (e.g., SNS topics, SQS queues, DDB tables) are visible to the application's lifecycle.
		suiteConf.appCtx = environment.Context()

		RunTestCaseApplication(t, suite, suiteConf, environment, func(app AppUnderTest) {
			method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
		})
	}, nil
}
