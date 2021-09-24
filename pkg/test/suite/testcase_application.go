package suite

import (
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	testCaseDefinitions["application"] = testCaseDefinition{
		matcher: isTestCaseApplication,
		builder: buildTestCaseApplication,
	}
}

func isTestCaseApplication(method reflect.Method) bool {
	if method.Func.Type().NumIn() != 2 {
		return false
	}

	if method.Func.Type().NumOut() != 0 {
		return false
	}

	actualType1 := method.Func.Type().In(1)
	expectedType := reflect.TypeOf((*AppUnderTest)(nil)).Elem()

	return actualType1 == expectedType
}

func buildTestCaseApplication(suite TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		suite.SetT(t)

		runTestCaseApplication(t, suite, suiteOptions, environment, func(aut *appUnderTest) {
			method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(aut)})
		})
	}, nil
}

func runTestCaseApplication(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment, testcase func(aut *appUnderTest)) {
	appOptions := append(suiteOptions.appOptions, []application.Option{
		application.WithProducerDaemon,
		application.WithConfigMap(map[string]interface{}{
			"env": "test",
		}),
	}...)

	ctx := environment.Context()
	config := environment.Config()
	logger := environment.Logger()

	app, err := application.NewWithInterfaces(ctx, config, logger, appOptions...)
	if err != nil {
		assert.FailNow(t, "failed to create application under test", err.Error())

		return
	}

	for name, module := range suiteOptions.appModules {
		app.Add(name, module)
	}

	for _, factory := range suiteOptions.appFactories {
		app.AddFactory(factory)
	}

	done := make(chan struct{})
	appDone := func() {
		<-done
	}

	appUnderTest := newAppUnderTest(app, appDone)

	go func() {
		app.Run()
		close(done)
	}()

	<-app.Running()
	testcase(appUnderTest)
	app.Stop("test done")
}
