package suite

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	RegisterTestCaseDefinition("application", isTestCaseApplication, buildTestCaseApplication)
}

const expectedTestCaseApplicationSignature = "func (s TestingSuite) TestFunc(AppUnderTest)"

func isTestCaseApplication(_ TestingSuite, method reflect.Method) error {
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

func buildTestCaseApplication(_ TestingSuite, method reflect.Method) (TestCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteConf *SuiteConfiguration, environment *env.Environment) {
		suite.SetT(t)

		RunTestCaseApplication(t, suite, suiteConf, environment, func(aut AppUnderTest) {
			method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(aut)})
		})
	}, nil
}

func RunTestCaseApplication(t *testing.T, _ TestingSuite, suiteConf *SuiteConfiguration, environment *env.Environment, testcase func(aut AppUnderTest), extraOptions ...Option) {
	var appOptions []application.Option

	for _, opt := range extraOptions {
		opt(suiteConf)
	}

	for k, factory := range suiteConf.appModules {
		suiteConf.appModules[k] = newEssentialModuleFactory(factory)
	}

	appOptions = append(appOptions, suiteConf.appOptions...)
	appOptions = append(appOptions, []application.Option{
		application.WithConfigMap(map[string]any{
			"app": map[string]any{
				"env": "test",
			},
		}),
		application.WithProducerDaemon,
		application.WithKernelExitHandler(func(code int) {
			assert.Equal(t, kernel.ExitCodeOk, code, "exit code should be %d", kernel.ExitCodeOk)
		}),
		application.WithMiddlewareFactory(reslife.LifeCycleManagerMiddleware, kernel.PositionBeginning),
		application.WithFixtureSetPostProcessorFactories(suiteConf.fixtureSetPostProcessorFactories...),
	}...)

	for _, factory := range suiteConf.fixtureSetFactories {
		appOptions = append(appOptions, application.WithFixtureSetFactory("default", factory))
	}

	for name, module := range suiteConf.appModules {
		appOptions = append(appOptions, application.WithModuleFactory(name, module))
	}

	for _, factory := range suiteConf.appFactories {
		appOptions = append(appOptions, application.WithModuleMultiFactory(factory))
	}

	config := environment.Config()
	logger := environment.Logger()

	// We need to create a new context here to isolate the individual apps from each other.
	// Else they would share the same container and module instances which can lead to issues.
	appCtx := appctx.WithContainer(t.Context())
	app, err := application.NewWithInterfaces(appCtx, config, logger, appOptions...)
	if err != nil {
		assert.FailNow(t, "failed to create application under test", err.Error())

		return
	}

	done := make(chan struct{})
	appDone := func() {
		<-done
	}

	appUnderTest := newAppUnderTest(appCtx, app, appDone)

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
