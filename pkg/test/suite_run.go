package test

import (
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
	"reflect"
	"regexp"
	"testing"
)

type AppDone func()

func RunCase(t *testing.T, suite TestingSuite) {
	suite.SetT(t)

	methodFinder := reflect.TypeOf(suite)

	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)

		if ok := filterTestMethod(t, method); !ok {
			continue
		}

		runCaseTest(t, suite, method)
	}
}

func runCaseTest(t *testing.T, suite TestingSuite, method reflect.Method) {
	suiteOptions := &suiteOptions{}

	for _, opt := range suite.SetupSuite() {
		opt(suiteOptions)
	}

	envOptions := []env.Option{
		env.WithLoggerSettingsFromConfig,
	}
	envOptions = append(envOptions, suiteOptions.envOptions...)

	environment, err := env.NewEnvironment(t, envOptions...)

	defer func() {
		if err = environment.Stop(); err != nil {
			assert.FailNow(t, "failed to stop test environment", err.Error())
		}
	}()

	if err != nil {
		assert.FailNow(t, "failed to create test environment", err.Error())
	}

	suite.SetEnv(environment)
	for _, envSetup := range suiteOptions.envSetup {
		envSetup()
	}

	appOptions := environment.ApplicationOptions()
	appOptions = append(suiteOptions.appOptions, appOptions...)
	appOptions = append(appOptions, []application.Option{
		application.WithConfigMap(map[string]interface{}{
			"env": "test",
		}),
	}...)

	app := application.New(appOptions...)

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

	go func() {
		app.Run()
		close(done)
	}()

	<-app.Running()

	method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(appDone)})

	app.Stop("test done")
}

func filterTestMethod(t *testing.T, method reflect.Method) bool {
	if ok, _ := regexp.MatchString("^Test", method.Name); !ok {
		return false
	}

	if method.Func.Type().NumIn() != 2 {
		assert.FailNow(t, "invalid test func signature", "test func %s has to have the signature func(wait test.AppDone)", method.Name)
	}

	arg1 := method.Func.Type().In(1)

	if arg1.Kind() != reflect.Func {
		assert.FailNow(t, "invalid test func signature", "test func %s has to have the signature func(wait test.AppDone)", method.Name)
	}

	if arg1.NumIn() > 0 {
		assert.FailNow(t, "invalid test func signature", "test func %s has to have the signature func(wait test.AppDone)", method.Name)
	}

	if arg1.NumOut() > 0 {
		assert.FailNow(t, "invalid test func signature", "test func %s has to have the signature func(wait test.AppDone)", method.Name)
	}

	return true

}
