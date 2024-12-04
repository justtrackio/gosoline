package suite

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	registerTestCaseDefinition("application", isTestCaseApplication, buildTestCaseApplication)
}

const expectedTestCaseApplicationSignature = "func (s TestingSuite) TestFunc(AppUnderTest)"

func isTestCaseApplication(method reflect.Method) error {
	if method.Func.Type().NumIn() != 2 {
		return fmt.Errorf("expected %q, but function has %d arguments", expectedTestCaseApplicationSignature, method.Func.Type().NumIn())
	}

	if method.Func.Type().NumOut() != 0 {
		return fmt.Errorf("expected %q, but function has %d return values", expectedTestCaseApplicationSignature, method.Func.Type().NumOut())
	}

	actualType0 := method.Func.Type().In(0)
	expectedType0 := reflect.TypeOf((*TestingSuite)(nil)).Elem()

	if !actualType0.Implements(expectedType0) {
		return fmt.Errorf("expected %q, but first argument type/receiver type is %s", expectedTestCaseApplicationSignature, actualType0.String())
	}

	actualType1 := method.Func.Type().In(1)
	expectedType1 := reflect.TypeOf((*AppUnderTest)(nil)).Elem()

	if actualType1 != expectedType1 {
		return fmt.Errorf("expected %q, but last argument type is %s", expectedTestCaseApplicationSignature, actualType1.String())
	}

	return nil
}

func buildTestCaseApplication(_ TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		suite.SetT(t)

		runTestCaseApplication(t, suite, suiteOptions, environment, func(aut *appUnderTest) {
			method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(aut)})
		})
	}, nil
}

func runTestCaseApplication(t *testing.T, _ TestingSuite, suiteOptions *suiteOptions, environment *env.Environment, testcase func(aut *appUnderTest)) {
	var appOptions []application.Option

	for k, factory := range suiteOptions.appModules {
		suiteOptions.appModules[k] = newEssentialModuleFactory(factory)
	}

	appOptions = append(appOptions, suiteOptions.appOptions...)
	appOptions = append(appOptions, []application.Option{
		application.WithConfigMap(map[string]any{
			"env": "test",
		}),
		application.WithProducerDaemon,
		application.WithKernelExitHandler(func(code int) {
			assert.Equal(t, kernel.ExitCodeOk, code, "exit code should be %d", kernel.ExitCodeOk)
		}),
	}...)

	for name, module := range suiteOptions.appModules {
		appOptions = append(appOptions, application.WithModuleFactory(name, module))
	}

	for _, factory := range suiteOptions.appFactories {
		appOptions = append(appOptions, application.WithModuleMultiFactory(factory))
	}

	config := environment.Config()
	logger := environment.Logger()

	app, err := application.NewWithInterfaces(environment.Context(), config, logger, appOptions...)
	if err != nil {
		assert.FailNow(t, "failed to create application under test", err.Error())

		return
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

	select {
	case <-app.Running():
		testcase(appUnderTest)
		app.Stop("test done")
	case <-done:
		app.Stop("app stopped")
	}
}

type essentialModule struct {
	kernel.Module
	kernel.EssentialModule
}

func (e essentialModule) IsHealthy(ctx context.Context) (bool, error) {
	if healthChecked, ok := e.Module.(kernel.HealthCheckedModule); ok {
		return healthChecked.IsHealthy(ctx)
	}

	return true, nil
}

func newEssentialModule(mod kernel.Module) kernel.Module {
	return essentialModule{
		mod,
		kernel.EssentialModule{},
	}
}

func newEssentialModuleFactory(factory kernel.ModuleFactory) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		mod, err := factory(ctx, config, logger)
		if err != nil {
			return nil, err
		}

		return newEssentialModule(mod), nil
	}
}
