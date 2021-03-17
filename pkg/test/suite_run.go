package test

import (
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
	"reflect"
	"regexp"
	"testing"
)

func RunSuite(t *testing.T, suite TestingSuite) {
	suite.SetT(t)

	methodFinder := reflect.TypeOf(suite)

	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)

		if ok := filterTestMethod(t, method); !ok {
			continue
		}

		RunTestCase(t, suite, func(appUnderTest AppUnderTest) {
			method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(appUnderTest)})
		})
	}
}

func filterTestMethod(t *testing.T, method reflect.Method) bool {
	if ok, _ := regexp.MatchString("^Test", method.Name); !ok {
		return false
	}

	if method.Func.Type().NumIn() != 2 {
		assert.FailNow(t, "invalid test func signature", "test func %s has to have the signature func(app test.AppUnderTest)", method.Name)
	}

	arg1 := method.Func.Type().In(1)

	if arg1 != reflect.TypeOf((*AppUnderTest)(nil)).Elem() {
		assert.FailNow(t, "invalid test func signature", "test func %s has to have the signature func(app test.AppUnderTest)", method.Name)
	}

	return true
}

func RunTestCase(t *testing.T, suite TestingSuite, testCase func(appUnderTest AppUnderTest), extraOptions ...SuiteOption) {
	suiteOptions := &suiteOptions{}

	setupOptions := []SuiteOption{
		WithClockProvider(clock.NewFakeClock()),
	}
	setupOptions = append(setupOptions, suite.SetupSuite()...)
	setupOptions = append(setupOptions, extraOptions...)

	for _, opt := range setupOptions {
		opt(suiteOptions)
	}

	envOptions := []env.Option{
		env.WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		env.WithLoggerSettingsFromConfig,
	}
	envOptions = append(envOptions, suiteOptions.envOptions...)
	envOptions = append(envOptions, env.WithConfigMap(map[string]interface{}{
		"env": "test",
	}))

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
		if err = envSetup(); err != nil {
			assert.FailNow(t, "failed to execute additional environment setup", err.Error())
		}
	}

	appOptions := environment.ApplicationOptions()
	appOptions = append(suiteOptions.appOptions, appOptions...)
	appOptions = append(appOptions, []application.Option{
		application.WithProducerDaemon,
		application.WithConfigMap(map[string]interface{}{
			"env": "test",
		}),
	}...)

	config := environment.Config()
	logger := environment.Logger()

	app, err := application.NewWithInterfaces(config, logger, appOptions...)

	if err != nil {
		assert.FailNow(t, "failed to create application under test", err.Error())
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

	testCase(appUnderTest)

	app.Stop("test done")
}
