package application_test

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
)

type testSettings struct {
	Field string `cfg:"field" default:"def"`
}

type testModule struct {
	kernel.EssentialModule
	t *testing.T
}

func (m testModule) Boot(config cfg.Config, _ log.Logger) error {
	settings := &testSettings{}
	if err := config.UnmarshalKey("test.settings-struct", settings); err != nil {
		return err
	}

	assert.Equal(m.t, "value", settings.Field)

	return nil
}

func (m testModule) Run(_ context.Context) error {
	return nil
}

func TestDefaultConfigParser(t *testing.T) {
	t.Setenv("TEST_SETTINGS_STRUCT_FIELD", "value")

	runTestApp(t, func() {
		config := application.WithConfigFile("config.dist.yml", "yml")
		exitCodeHandler := application.WithKernelExitHandler(func(code int) {})
		moduleOption := application.WithModuleFactory("test", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return testModule{
				t: t,
			}, nil
		})

		app := application.Default(config, exitCodeHandler, moduleOption)
		app.Run()
	})
}

func TestDefaultDoesNotInstallOtelShutdownHandlers(t *testing.T) {
	app := application.Default(
		application.WithKernelExitHandler(func(int) {}),
		application.WithModuleFactory("module", func(context.Context, cfg.Config, log.Logger) (kernel.Module, error) {
			return kernel.NewModuleFunc(func(context.Context) error { return nil }), nil
		}),
	)
	shutdownHandlers := reflect.ValueOf(app).Elem().FieldByName("shutdownHandlers")

	assert.Zero(t, shutdownHandlers.Len())
}

func TestWithOtelShutdownRunsHandlersInOrder(t *testing.T) {
	metric.ResetShutdownRegistry()
	tracing.ResetShutdownRegistry()
	log.ResetShutdownRegistry()
	t.Cleanup(metric.ResetShutdownRegistry)
	t.Cleanup(tracing.ResetShutdownRegistry)
	t.Cleanup(log.ResetShutdownRegistry)

	order := []string{}
	metric.RegisterShutdown("metric", func(context.Context) error {
		order = append(order, "metric")

		return nil
	})
	tracing.RegisterShutdown("tracing", func(context.Context) error {
		order = append(order, "tracing")

		return nil
	})
	log.RegisterShutdown("log", func(context.Context) error {
		order = append(order, "log")

		return nil
	})

	app := application.New(
		application.WithOtelShutdown,
		application.WithKernelExitHandler(func(int) {}),
		application.WithModuleFactory("module", func(context.Context, cfg.Config, log.Logger) (kernel.Module, error) {
			return kernel.NewModuleFunc(func(context.Context) error { return nil }), nil
		}),
	)
	app.Run()

	assert.Equal(t, []string{"metric", "tracing", "log"}, order)
}

func runTestApp(t *testing.T, f func()) {
	oldDir, err := os.Getwd()
	assert.NoError(t, err)

	t.Chdir("testdata")

	defer func() {
		t.Chdir(oldDir)
	}()

	args := os.Args
	os.Args = []string{os.Args[0]}
	defer func() {
		os.Args = args
		assert.Nil(t, recover(), "App should not fail to be created")
	}()

	f()
}
