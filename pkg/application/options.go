package application

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	taskRunner "github.com/justtrackio/gosoline/pkg/conc/task_runner"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	kernelPkg "github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/metric/calculator"
	"github.com/justtrackio/gosoline/pkg/share"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/pkg/errors"
)

type (
	Option       func(app *App)
	ConfigOption func(config cfg.GosoConf) error
	LoggerOption func(config cfg.GosoConf, logger log.GosoLogger) error
	KernelOption func(config cfg.GosoConf) kernelPkg.Option
	SetupOption  func(ctx context.Context, config cfg.GosoConf, logger log.GosoLogger) error
)

func WithHttpHealthCheck(app *App) {
	WithModuleFactory("http-health-check", httpserver.NewHealthCheck())(app)
}

func WithConfigEnvKeyPrefix(prefix string) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			prefix = strings.ReplaceAll(prefix, "-", "_")

			return config.Option(cfg.WithEnvKeyPrefix(prefix))
		})
	}
}

func WithConfigEnvKeyReplacer(replacer *strings.Replacer) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			if err := config.Option(cfg.WithEnvKeyReplacer(replacer)); err != nil {
				return err
			}

			return nil
		})
	}
}

func WithConfigCallback(call func(config cfg.GosoConf) error) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return call(config)
		})
	}
}

func WithConfigDebug(app *App) {
	app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
		return kernelPkg.WithMiddlewareFactory(func(ctx context.Context, config cfg.Config, logger log.Logger) (kernelPkg.Middleware, error) {
			return func(next kernelPkg.MiddlewareHandler) kernelPkg.MiddlewareHandler {
				return func() {
					if err := cfg.DebugConfig(config, logger); err != nil {
						logger.Error("can not debug config: %w", err)
					}

					next()
				}
			}, nil
		}, kernelPkg.PositionEnd)
	})
}

func WithConfigErrorHandlers(handlers ...cfg.ErrorHandler) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithErrorHandlers(handlers...))
		})
	}
}

func WithConfigBytes(bytes []byte, format string) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigBytes(bytes, format))
		})
	}
}

func WithConfigFile(filePath string, fileType string) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigFile(filePath, fileType))
		})
	}
}

func WithConfigFlags(args []string, opts any) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			if _, err := flags.NewParser(opts, flags.IgnoreUnknown).ParseArgs(args); err != nil {
				return fmt.Errorf("can not parse command line flags: %w", err)
			}

			st, err := mapx.NewStruct(opts, &mapx.StructSettings{
				FieldTag:   "cfg",
				DefaultTag: "default",
			})
			if err != nil {
				return fmt.Errorf("can create mapx from flag struct: %w", err)
			}

			mpx, err := st.Read()
			if err != nil {
				return fmt.Errorf("can read flag struct with mapx: %w", err)
			}

			msi := mpx.Msi()
			if err = config.Option(cfg.WithConfigMap(msi)); err != nil {
				return fmt.Errorf("can not set config map: %w", err)
			}

			return nil
		})
	}
}

func WithConfigFileFlag(app *App) {
	app.addConfigOption(func(config cfg.GosoConf) error {
		var opts struct {
			Config []string `long:"config" short:"c"`
		}

		if _, err := flags.NewParser(&opts, flags.IgnoreUnknown).ParseArgs(os.Args); err != nil {
			return fmt.Errorf("can not parse command line flags: %w", err)
		}

		var options []cfg.Option
		for _, configFile := range opts.Config {
			options = append(options, cfg.WithConfigFile(configFile, "yml"))
		}

		return config.Option(options...)
	})
}

func WithConfigMap(configMap map[string]any, mergeOptions ...cfg.MergeOption) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigMap(configMap, mergeOptions...))
		})
	}
}

func WithConfigPostProcessor(processor cfg.PostProcessor) Option {
	return func(app *App) {
		app.configPostProcessors = append(app.configPostProcessors, processor)
	}
}

func WithConfigSanitizers(sanitizers ...cfg.Sanitizer) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithSanitizers(sanitizers...))
		})
	}
}

func WithConfigSetting(key string, settings any) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigSetting(key, settings))
		})
	}
}

func WithMetricsCalculatorModule(app *App) {
	app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
		return kernelPkg.WithModuleMultiFactory(calculator.CalculatorModuleFactory)
	})
}

func WithExecBackoffInfinite(app *App) {
	app.addConfigOption(func(config cfg.GosoConf) error {
		return config.Option(cfg.WithConfigSetting("exec.backoff.type", "infinite"))
	})
}

func WithExecBackoffSettings(settings *exec.BackoffSettings) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigSetting("exec.backoff", settings, cfg.SkipExisting))
		})
	}
}

func WithDbRepoChangeHistory(app *App) {
	app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
		return kernelPkg.WithMiddlewareFactory(db_repo.KernelMiddlewareChangeHistory, kernelPkg.PositionEnd)
	})
}

func WithHttpServerShares(app *App) {
	app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
		return kernelPkg.WithMiddlewareFactory(share.KernelMiddlewareShares, kernelPkg.PositionEnd)
	})
}

func WithFixtureSetFactory(group string, factory fixtures.FixtureSetsFactory) Option {
	return func(app *App) {
		app.addSetupOption(func(ctx context.Context, config cfg.GosoConf, logger log.GosoLogger) error {
			return fixtures.AddFixtureSetFactory(ctx, group, factory)
		})
	}
}

func WithFixtureSetFactories(factories map[string]fixtures.FixtureSetsFactory) Option {
	return func(app *App) {
		app.addSetupOption(func(ctx context.Context, config cfg.GosoConf, logger log.GosoLogger) error {
			for group, factory := range factories {
				if err := fixtures.AddFixtureSetFactory(ctx, group, factory); err != nil {
					return err
				}
			}

			return nil
		})
	}
}

func WithFixtureSetPostProcessorFactories(factories ...fixtures.PostProcessorFactory) Option {
	return func(app *App) {
		app.addSetupOption(func(ctx context.Context, config cfg.GosoConf, logger log.GosoLogger) error {
			return fixtures.AddFixtureSetPostProcessorFactory(ctx, factories...)
		})
	}
}

func WithKernelExitHandler(handler kernelPkg.ExitHandler) Option {
	return func(app *App) {
		app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
			return kernelPkg.WithExitHandler(handler)
		})
	}
}

func WithKinsumerAutoscaleModule(kinsumerInputName string) Option {
	return func(app *App) {
		app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
			return kernelPkg.WithModuleMultiFactory(stream.KinsumerAutoscaleModuleFactory(kinsumerInputName))
		})
	}
}

func WithLoggerGroupTag(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
		if !config.IsSet("app_group") {
			return errors.New("can not get application group from config to set it on logger")
		}

		appGroup, err := config.GetString("app_group")
		if err != nil {
			return fmt.Errorf("failed to get app_group config: %w", err)
		}

		return logger.Option(log.WithFields(map[string]any{
			"group": appGroup,
		}))
	})
}

func WithLoggerApplicationTag(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
		if !config.IsSet("app_name") {
			return errors.New("can not get application name from config to set it on logger")
		}

		appName, err := config.GetString("app_name")
		if err != nil {
			return fmt.Errorf("failed to get app_name config: %w", err)
		}

		return logger.Option(log.WithFields(map[string]any{
			"application": appName,
		}))
	})
}

func WithLoggerContextFieldsMessageEncoder(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
		stream.AddDefaultEncodeHandler(log.NewMessageWithLoggingFieldsEncoder(config, logger))

		return nil
	})
}

func WithLoggerContextFieldsResolver(resolver ...log.ContextFieldsResolverFunction) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
			return logger.Option(log.WithContextFieldsResolver(resolver...))
		})
	}
}

func WithLoggerHandlers(handler ...log.Handler) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
			return logger.Option(log.WithHandlers(handler...))
		})
	}
}

func WithLoggerHandlersFromConfig(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
		var err error
		var handlers []log.Handler

		if handlers, err = log.NewHandlersFromConfig(config); err != nil {
			return fmt.Errorf("can not create handlers from config: %w", err)
		}

		return logger.Option(log.WithHandlers(handlers...))
	})
}

func WithLoggerMetricHandler(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
		metricHandler := metric.NewLoggerHandler()

		return logger.Option(log.WithHandlers(metricHandler))
	})
}

func WithLoggerSentryHandler(contextProvider ...log.SentryContextProvider) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
			var err error
			var sentryHandler *log.HandlerSentry

			if sentryHandler, err = log.NewHandlerSentry(config); err != nil {
				return fmt.Errorf("can not create logger sentry handler: %w", err)
			}

			for _, provider := range contextProvider {
				if err = provider(config, sentryHandler); err != nil {
					return fmt.Errorf("can not run sentry context provider %T: %w", provider, err)
				}
			}

			return logger.Option(log.WithHandlers(sentryHandler))
		})
	}
}

func WithMetadataServer(app *App) {
	WithModuleFactory("metadata-server", NewMetadataServer)(app)
}

func WithMetrics(app *App) {
	WithModuleFactory("metric-daemon", metric.NewDaemonModule)(app)
	WithModuleFactory("prometheus-metrics-server", metric.NewPrometheusMetricsServerModule)(app)
}

func WithProducerDaemon(app *App) {
	app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
		return kernelPkg.WithModuleMultiFactory(stream.ProducerDaemonFactory)
	})
}

func WithTaskRunner(app *App) {
	app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
		return kernelPkg.WithModuleMultiFactory(taskRunner.Factory)
	})
}

func WithProfiling(app *App) {
	app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
		return kernelPkg.WithModuleMultiFactory(httpserver.ProfilingModuleFactory)
	})
}

func WithTracing(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger log.GosoLogger) error {
		tracingHandler := tracing.NewLoggerErrorHandler()

		options := []log.Option{
			log.WithHandlers(tracingHandler),
			log.WithContextFieldsResolver(tracing.ContextTraceFieldsResolver),
		}

		return logger.Option(options...)
	})

	app.addSetupOption(func(ctx context.Context, config cfg.GosoConf, logger log.GosoLogger) error {
		strategy := tracing.NewTraceIdErrorWarningStrategy(logger)
		stream.AddDefaultEncodeHandler(tracing.NewMessageWithTraceEncoder(strategy))

		return nil
	})
}

func WithMiddlewareFactory(factory kernelPkg.MiddlewareFactory, position kernelPkg.Position) Option {
	return func(app *App) {
		app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
			return kernelPkg.WithMiddlewareFactory(factory, position)
		})
	}
}

func WithModuleFactory(name string, moduleFactory kernelPkg.ModuleFactory, opts ...kernelPkg.ModuleOption) Option {
	return func(app *App) {
		app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
			return kernelPkg.WithModuleFactory(name, moduleFactory, opts...)
		})
	}
}

func WithModuleMultiFactory(factory kernelPkg.ModuleMultiFactory) Option {
	return func(app *App) {
		app.addKernelOption(func(config cfg.GosoConf) kernelPkg.Option {
			return kernelPkg.WithModuleMultiFactory(factory)
		})
	}
}

func WithUTCClock(useUTC bool) Option {
	return func(app *App) {
		app.addSetupOption(func(ctx context.Context, config cfg.GosoConf, logger log.GosoLogger) error {
			clock.WithUseUTC(useUTC)

			return nil
		})
	}
}
