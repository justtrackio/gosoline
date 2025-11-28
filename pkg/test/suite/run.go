package suite

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

type (
	TestCaseMatcher func(suite TestingSuite, method reflect.Method) error
	TestCaseBuilder func(suite TestingSuite, method reflect.Method) (TestCaseRunner, error)
	TestCaseRunner  func(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, environment *env.Environment)
)

type testCaseDefinition struct {
	matcher TestCaseMatcher
	builder TestCaseBuilder
}

var testCaseDefinitions = map[string]testCaseDefinition{}

// Register a new test case definition for our test suite. A definition consists of a name, a matcher, and a builder.
//   - The name of a test case definition is used when reporting errors in case something goes wrong. It has to be unique.
//   - The matcher is called on every method of a test suite starting with "Test" and should check if the function has the
//     correct argument and return types. If there is any mismatch between the expected and actual types, an error has to
//     be reported. If a matcher from another test case definition matches the method, that definition will be used to execute
//     the test, if no matcher successfully matches the test case, an error is reported.
//   - The builder creates a runner for the test from the matched method. It might execute the method to get some kind of
//     test case definition (http, consumer, and subscriber test cases work like this) which is later used to run the test.
//     Or the execution of the method might be part of running the actual test (base and application test cases work like this).
//
// This function is not thread safe and should only be called from an init() method.
func RegisterTestCaseDefinition(name string, matcher TestCaseMatcher, builder TestCaseBuilder) {
	if _, ok := testCaseDefinitions[name]; ok {
		panic(fmt.Sprintf("test case definition %q was already registered", name))
	}

	testCaseDefinitions[name] = testCaseDefinition{
		matcher: matcher,
		builder: builder,
	}
}

func Run(t *testing.T, suite TestingSuite, extraOptions ...Option) {
	suite.SetT(t)

	var err error
	var testCases map[string]TestCaseRunner
	suiteConf := suiteConfApplyOptions(suite, extraOptions)

	if testCases, err = suiteFindTestCases(suite, suiteConf); err != nil {
		assert.FailNow(t, err.Error())

		return
	}

	if len(testCases) == 0 {
		return
	}

	for i := 0; i < suiteConf.testCaseRepeatCount; i++ {
		if suiteConf.envIsShared {
			runTestCaseWithSharedEnvironment(t, suite, suiteConf, testCases)
		} else {
			runTestCaseWithIsolatedEnvironment(t, suite, suiteConf, testCases)
		}
	}
}

func suiteConfApplyOptions(suite TestingSuite, extraOptions []Option) *SuiteConfiguration {
	options := []Option{
		WithClockProvider(clock.NewFakeClock()),
		WithConfigMap(map[string]any{
			"cloud.aws.default.ec2.metadata.available": false,
			"kernel": map[string]any{
				"health_check": map[string]any{
					"wait_interval": "10ms",
				},
			},
		}),
	}
	options = append(options, suite.SetupSuite()...)
	options = append(options, extraOptions...)

	return newSuiteConfiguration(options)
}

func suiteFindTestCases(suite TestingSuite, conf *SuiteConfiguration) (map[string]TestCaseRunner, error) {
	var err error
	testCases := make(map[string]TestCaseRunner)
	methodFinder := reflect.TypeOf(suite)

	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)

		if !strings.HasPrefix(method.Name, "Test") {
			continue
		}

		var matcherErr *multierror.Error
		for typ, def := range testCaseDefinitions {
			if err := def.matcher(suite, method); err != nil {
				matcherErr = multierror.Append(matcherErr, fmt.Errorf("matcher for test case type %s failed: %w", typ, err))

				continue
			}

			matcherErr = nil

			if conf.shouldSkip(method.Name) {
				testCases[method.Name] = func(t *testing.T, _ TestingSuite, _ *SuiteConfiguration, _ *env.Environment) {
					t.SkipNow()
				}

				break
			}

			if testCases[method.Name], err = def.builder(suite, method); err != nil {
				return nil, fmt.Errorf("can not build test case %s of type %s: %w", method.Name, typ, err)
			}

			break
		}

		if err := matcherErr.ErrorOrNil(); err != nil {
			assert.Fail(suite.T(), fmt.Sprintf("test method %q has wrong signature: %s", method.Name, err.Error()))
		}
	}

	if len(testCases) == 0 {
		return nil, fmt.Errorf("no testcases found. the function name has to start with 'Test'")
	}

	return testCases, nil
}

func runTestCaseWithSharedEnvironment(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, testCases map[string]TestCaseRunner) {
	envOptions := []env.Option{
		env.WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		env.WithConfigMap(map[string]any{
			"app_project": "justtrack",
			"app_family":  "gosoline",
			"app_group":   "test",
			"app_name":    "test",
		}),
	}
	envOptions = append(envOptions, suiteConf.envOptions...)
	envOptions = append(envOptions, env.WithConfigMap(map[string]any{
		"env":              "test",
		"fixtures.enabled": true,
		"resource_lifecycles": map[string]any{
			"purge": map[string]any{
				"enabled": true,
			},
		},
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

	suite.SetEnv(environment)

	for _, envSetup := range suiteConf.envSetup {
		if err := envSetup(); err != nil {
			assert.FailNow(t, "failed to execute additional environment setup", err.Error())
		}
	}

	for name, testCase := range testCases {
		RunTestCaseInSuite(t, suite, func() {
			t.Run(name, func(t *testing.T) {
				parentT := suite.T()
				suite.SetT(t)
				defer suite.SetT(parentT)

				testCase(t, suite, suiteConf, environment)
			})
		})
	}
}

func runTestCaseWithIsolatedEnvironment(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, testCases map[string]TestCaseRunner) {
	for name, testCase := range testCases {
		runTestCaseWithSharedEnvironment(t, suite, suiteConf, map[string]TestCaseRunner{
			name: testCase,
		})
	}
}

func RunTestCaseInSuite(t *testing.T, suite TestingSuite, testCase func()) {
	parentT := suite.T()
	suite.SetT(t)
	defer suite.SetT(parentT)

	if setupTestAware, ok := suite.(TestingSuiteSetupTestAware); ok {
		if err := setupTestAware.SetupTest(); err != nil {
			assert.FailNow(suite.T(), "failed to setup the test", err.Error())
		}
	}

	// defer the cleanup so it also gets called when we skip the test
	defer func() {
		if tearDownTestAware, ok := suite.(TestingSuiteTearDownTestAware); ok {
			if err := tearDownTestAware.TearDownTest(); err != nil {
				assert.FailNow(suite.T(), "failed to tear down the test", err.Error())
			}
		}

		stream.ResetInMemoryInputs()
		stream.ResetInMemoryOutputs()
		suite.Env().ResetLogs()
	}()

	testCase()
}
