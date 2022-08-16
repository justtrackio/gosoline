package suite

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/justtrackio/resty/v2"
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

func isTestCaseApiServer(method reflect.Method) bool {
	if method.Func.Type().NumIn() != 3 {
		return false
	}

	if method.Func.Type().NumOut() != 1 {
		return false
	}

	actualType1 := method.Func.Type().In(1)
	expectedType1 := reflect.TypeOf((*AppUnderTest)(nil)).Elem()

	if actualType1 != expectedType1 {
		return false
	}

	actualType2 := method.Func.Type().In(2)
	expectedType2 := reflect.TypeOf(&resty.Client{})

	return actualType2 == expectedType2
}

func buildTestCaseApiServer(suite TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return runTestCaseApiServer(suite, func(suite TestingSuite, app *appUnderTest, client *resty.Client) {
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

func runTestCaseApiServer(suite TestingSuite, testCase func(suite TestingSuite, app *appUnderTest, client *resty.Client)) (testCaseRunner, error) {
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

			url := fmt.Sprintf("http://127.0.0.1:%d", *port)
			client := resty.New().SetBaseURL(url)

			testCase(suite, app, client)
		})
	}, nil
}
