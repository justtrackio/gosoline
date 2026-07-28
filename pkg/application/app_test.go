package application_test

import (
	"context"
	"os"
	"testing"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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

func TestWithOtelShutdownRunsHandlersInOrder(t *testing.T) {
	order := []string{}

	app := application.New(
		application.WithOtelShutdown,
		application.WithKernelExitHandler(func(int) {}),
		application.WithModuleFactory("module", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			metricShutdownHandler, err := metric.ProvideShutdownHandler(ctx, config, logger)
			if err != nil {
				return nil, err
			}
			metricShutdownHandler.AddProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(&metricShutdownExporter{order: &order}))))

			tracingShutdownHandler, err := tracing.ProvideShutdownHandler(ctx, config, logger)
			if err != nil {
				return nil, err
			}
			tracingShutdownHandler.AddProvider(sdktrace.NewTracerProvider(sdktrace.WithSyncer(&tracingShutdownExporter{order: &order})))

			return kernel.NewModuleFunc(func(context.Context) error { return nil }), nil
		}),
	)
	app.Run()

	assert.Equal(t, []string{"metric", "tracing"}, order)
}

type metricShutdownExporter struct {
	order *[]string
}

func (e *metricShutdownExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *metricShutdownExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *metricShutdownExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	return nil
}

func (e *metricShutdownExporter) ForceFlush(context.Context) error {
	return nil
}

func (e *metricShutdownExporter) Shutdown(context.Context) error {
	*e.order = append(*e.order, "metric")

	return nil
}

type tracingShutdownExporter struct {
	order *[]string
}

func (e *tracingShutdownExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *tracingShutdownExporter) Shutdown(context.Context) error {
	*e.order = append(*e.order, "tracing")

	return nil
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
