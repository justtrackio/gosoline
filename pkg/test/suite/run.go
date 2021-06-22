package suite

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strings"
	"testing"
)

type testCaseMatcher func(method reflect.Method) bool
type testCaseBuilder func(suite TestingSuite, method reflect.Method) (testCaseRunner, error)
type testCaseRunner func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment)

type testCaseDefinition struct {
	matcher testCaseMatcher
	builder testCaseBuilder
}

var testCaseDefinitions = map[string]testCaseDefinition{}

func Run(t *testing.T, suite TestingSuite, extraOptions ...Option) {
	var err error
	var testCases map[string]testCaseRunner
	var suiteOptions = suiteApplyOptions(suite, extraOptions)

	if testCases, err = suiteFindTestCases(t, suite); err != nil {
		assert.FailNow(t, err.Error())
		return
	}

	if len(testCases) == 0 {
		return
	}

	if suiteOptions.envIsShared {
		runTestCaseWithSharedEnvironment(t, suite, suiteOptions, testCases)
	} else {
		runTestCaseWithIsolatedEnvironment(t, suite, suiteOptions, testCases)
	}
}

func suiteFindTestCases(_ *testing.T, suite TestingSuite) (map[string]testCaseRunner, error) {
	var err error
	var testCases = make(map[string]testCaseRunner)
	var methodFinder = reflect.TypeOf(suite)

	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)

		if !strings.HasPrefix(method.Name, "Test") {
			continue
		}

		for typ, def := range testCaseDefinitions {
			if !def.matcher(method) {
				continue
			}

			if testCases[method.Name], err = def.builder(suite, method); err != nil {
				return nil, fmt.Errorf("can not build test case %s of type %s: %w", method.Name, typ, err)
			}
		}
	}

	return testCases, nil
}

func suiteApplyOptions(suite TestingSuite, extraOptions []Option) *suiteOptions {
	setupOptions := []Option{
		WithClockProvider(clock.NewFakeClock()),
	}
	setupOptions = append(setupOptions, suite.SetupSuite()...)
	setupOptions = append(setupOptions, extraOptions...)

	options := newSuiteOptions()

	for _, opt := range setupOptions {
		opt(options)
	}

	return options
}

func runTestCaseWithSharedEnvironment(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, testCases map[string]testCaseRunner) {
	envOptions := []env.Option{
		env.WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
	}
	envOptions = append(envOptions, suiteOptions.envOptions...)
	envOptions = append(envOptions, env.WithConfigMap(map[string]interface{}{
		"env": "test",
	}))

	environment, err := env.NewEnvironment(t, envOptions...)

	if err != nil {
		assert.FailNow(t, "failed to create test environment", err.Error())
	}

	defer func() {
		if err := environment.Stop(); err != nil {
			assert.FailNow(t, "failed to stop test environment", err.Error())
		}
	}()

	suite.SetT(t)
	suite.SetEnv(environment)

	for _, envSetup := range suiteOptions.envSetup {
		if err := envSetup(); err != nil {
			assert.FailNow(t, "failed to execute additional environment setup", err.Error())
		}
	}

	for name, testCase := range testCases {
		if setupTestAware, ok := suite.(TestingSuiteSetupTestAware); ok {
			if err := setupTestAware.SetupTest(); err != nil {
				assert.FailNow(t, "failed to setup the test", err.Error())
			}
		}

		t.Run(name, func(t *testing.T) {
			testCase(t, suite, suiteOptions, environment)
		})

		if tearDownTestAware, ok := suite.(TestingSuiteTearDownTestAware); ok {
			if err := tearDownTestAware.TearDownTest(); err != nil {
				assert.FailNow(t, "failed to tear down the test", err.Error())
			}
		}

		stream.ResetInMemoryInputs()
		stream.ResetInMemoryOutputs()
		stream.ResetProducerDaemons()
		kvstore.ResetConfigurableKvStores()
	}
}

func runTestCaseWithIsolatedEnvironment(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, testCases map[string]testCaseRunner) {
	for name, testCase := range testCases {
		runTestCaseWithSharedEnvironment(t, suite, suiteOptions, map[string]testCaseRunner{
			name: testCase,
		})
	}
}
