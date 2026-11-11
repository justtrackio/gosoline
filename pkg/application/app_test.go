package application_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/smpl/smplctx"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWithLoggerSampling(t *testing.T) {
	for name, enabled := range map[string]bool{"enabled": true, "disabled": false} {
		t.Run(name, func(t *testing.T) {
			ctx := appctx.WithContainer(t.Context())
			ctx = smplctx.WithSampling(ctx, smplctx.Sampling{Sampled: false})
			before, err := stream.NewMessageEncoder(&stream.MessageEncoderSettings{}).Encode(ctx, "payload")
			require.NoError(t, err)

			config := cfg.New(map[string]any{
				"app":      map[string]any{"env": "test", "name": "sampling-test"},
				"sampling": map[string]any{"enabled": enabled},
			})
			buf := new(bytes.Buffer)
			handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterJson, "main", time.RFC3339, buf)
			logger := log.NewLoggerWithInterfaces(clock.NewFakeClock(), []log.Handler{handler})
			_, err = application.NewWithInterfaces(ctx, config, logger,
				application.WithLoggerSampling,
				application.WithModuleFactory("test", func(context.Context, cfg.Config, log.Logger) (kernel.Module, error) {
					return kernel.NewModuleFunc(func(context.Context) error { return nil }), nil
				}),
			)
			require.NoError(t, err)
			buf.Reset()

			logger.Info(ctx, "ordinary log")
			assert.Equal(t, enabled, buf.Len() == 0)

			after, err := stream.NewMessageEncoder(&stream.MessageEncoderSettings{}).Encode(ctx, "payload")
			require.NoError(t, err)
			assert.Equal(t, before.Attributes, after.Attributes)
		})
	}
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
