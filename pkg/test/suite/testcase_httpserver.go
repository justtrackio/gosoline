package suite

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	RegisterTestCaseDefinition("httpserver", isTestCaseHttpserver, buildTestCaseHttpserver)
}

const expectedTestCaseHttpserverSignature = "func (s TestingSuite) TestFunc(AppUnderTest, *resty.Client) error"

type TestingSuiteApiDefinitionsAware interface {
	SetupApiDefinitions() httpserver.Definer
}

func isTestCaseHttpserver(s TestingSuite, method reflect.Method) error {
	if _, ok := s.(TestingSuiteApiDefinitionsAware); !ok {
		return fmt.Errorf("the suite has to implement the TestingSuiteApiDefinitionsAware interface to be able to run httpserver test cases")
	}

	if method.Func.Type().NumIn() != 3 {
		return fmt.Errorf("expected %q, but function has %d arguments", expectedTestCaseHttpserverSignature, method.Func.Type().NumIn())
	}

	if method.Func.Type().NumOut() != 1 {
		return fmt.Errorf("expected %q, but function has %d return values", expectedTestCaseHttpserverSignature, method.Func.Type().NumOut())
	}

	actualType0 := method.Func.Type().In(0)
	expectedType0 := reflect.TypeOf((*TestingSuite)(nil)).Elem()

	if !actualType0.Implements(expectedType0) {
		return fmt.Errorf("expected %q, but first argument type/receiver type is %s", expectedTestCaseHttpserverSignature, actualType0.String())
	}

	actualType1 := method.Func.Type().In(1)
	expectedType1 := reflect.TypeOf((*AppUnderTest)(nil)).Elem()

	if actualType1 != expectedType1 {
		return fmt.Errorf("expected %q, but first argument type is %s", expectedTestCaseHttpserverSignature, actualType1.String())
	}

	actualType2 := method.Func.Type().In(2)
	expectedType2 := reflect.TypeOf((*resty.Client)(nil))

	if actualType2 != expectedType2 {
		return fmt.Errorf("expected %q, but last argument type is %s", expectedTestCaseHttpserverSignature, actualType2.String())
	}

	actualTypeResult := method.Func.Type().Out(0)
	expectedTypeResult := reflect.TypeOf((*error)(nil)).Elem()

	if actualTypeResult != expectedTypeResult {
		return fmt.Errorf("expected %q, but return type is %s", expectedTestCaseHttpserverSignature, actualTypeResult.String())
	}

	return nil
}

func buildTestCaseHttpserver(suite TestingSuite, method reflect.Method) (TestCaseRunner, error) {
	return runTestCaseHttpserver(suite, func(suite TestingSuite, app AppUnderTest, client *resty.Client) {
		out := method.Func.Call([]reflect.Value{
			reflect.ValueOf(suite),
			reflect.ValueOf(app),
			reflect.ValueOf(client),
		})

		result := out[0].Interface()

		if result == nil {
			return
		}

		if err := result.(error); err != nil {
			assert.FailNow(suite.T(), err.Error(), "testcase %s returned an unexpected error: %s", method.Name, err)
		}
	})
}

func runTestCaseHttpserver(suite TestingSuite, testCase func(suite TestingSuite, app AppUnderTest, client *resty.Client)) (TestCaseRunner, error) {
	var ok bool
	var apiDefinitionAware TestingSuiteApiDefinitionsAware
	var server *httpserver.HttpServer

	if apiDefinitionAware, ok = suite.(TestingSuiteApiDefinitionsAware); !ok {
		return nil, fmt.Errorf("the suite has to implement the TestingSuiteApiDefinitionsAware interface to be able to run httpserver test cases")
	}

	return func(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, environment *env.Environment) {
		// we first have to set up t, otherwise the test suite can't assert that there are no errors when setting up
		// route definitions or test cases
		suite.SetT(t)

		apiDefinitions := apiDefinitionAware.SetupApiDefinitions()

		suiteConf.appModules["api"] = func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			module, err := httpserver.New("default", apiDefinitions)(ctx, config, logger)
			if err != nil {
				return nil, fmt.Errorf("failed to create test http server: %w", err)
			}

			server = module.(*httpserver.HttpServer)

			return server, nil
		}

		suiteConf.addAppOption(application.WithConfigMap(map[string]any{
			"httpserver": map[string]any{
				"default": map[string]any{
					"port": 0,
				},
			},
		}))

		RunTestCaseApplication(t, suite, suiteConf, environment, func(app AppUnderTest) {
			port, err := server.GetPort()
			if err != nil {
				assert.FailNow(t, err.Error(), "can not get port of server")

				return
			}

			url := fmt.Sprintf("http://127.0.0.1:%d", *port)
			client := resty.New().SetBaseURL(url)

			testCase(suite, app, client)
		})
	}, nil
}
