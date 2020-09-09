package application

import (
	"flag"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/fixtures"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

type Option func(app *App)
type ConfigOption func(config cfg.GosoConf) error
type LoggerOption func(config cfg.GosoConf, logger mon.GosoLog) error
type KernelOption func(config cfg.GosoConf, kernel kernel.GosoKernel) error
type SetupOption func(config cfg.GosoConf, logger mon.GosoLog) error

type kernelSettings struct {
	KillTimeout time.Duration `cfg:"killTimeout" default:"10s"`
}

type loggerHandler struct {
	Name string `cfg:"type"`
}

type loggerSettings struct {
	Level           string                 `cfg:"level" default:"info" validate:"required"`
	Format          string                 `cfg:"format" default:"console" validate:"required"`
	TimestampFormat string                 `cfg:"timestamp_format" default:"15:04:05.000" validate:"required"`
	Tags            map[string]interface{} `cfg:"tags"`
}

func WithApiHealthCheck(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.GosoKernel) error {
		kernel.Add("api-health-check", apiserver.NewApiHealthCheck())
		return nil
	})
}

func WithConfigEnvKeyPrefix(prefix string) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			prefix = strings.Replace(prefix, "-", "_", -1)

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

func WithConfigErrorHandlers(handlers ...cfg.ErrorHandler) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithErrorHandlers(handlers...))
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

func WithConfigFileFlag(app *App) {
	app.addConfigOption(func(config cfg.GosoConf) error {
		flags := flag.NewFlagSet("cfg", flag.ContinueOnError)

		configFile := flags.String("config", "", "path to a config file")
		err := flags.Parse(os.Args[1:])

		if err != nil {
			return err
		}

		return config.Option(cfg.WithConfigFile(*configFile, "yml"))
	})
}

func WithConfigMap(configMap map[string]interface{}) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigMap(configMap))
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

func WithConfigServer(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.GosoKernel) error {
		kernel.Add("config-server", new(ConfigServer))
		return nil
	})
}

func WithConfigSetting(key string, settings interface{}) Option {
	return func(app *App) {
		app.addConfigOption(func(config cfg.GosoConf) error {
			return config.Option(cfg.WithConfigSetting(key, settings))
		})
	}
}

func WithFixtures(fixtureSets []*fixtures.FixtureSet) Option {
	return func(app *App) {
		app.addSetupOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			loader := fixtures.NewFixtureLoader(config, logger)
			return loader.Load(fixtureSets)
		})
	}
}

func WithKernelSettingsFromConfig(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, k kernel.GosoKernel) error {
		settings := &kernelSettings{}
		config.UnmarshalKey("kernel", settings)

		return k.Option(kernel.KillTimeout(settings.KillTimeout))
	})
}

func WithLoggerApplicationTag(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		if !config.IsSet("app_name") {
			return errors.New("can not get application name from config to set it on logger")
		}

		return logger.Option(mon.WithTags(map[string]interface{}{
			"application": config.GetString("app_name"),
		}))
	})
}

func WithLoggerContextFieldsMessageEncoder() Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			stream.AddDefaultEncodeHandler(mon.NewMessageWithLoggingFieldsEncoder(config, logger))
			return nil
		})
	}
}

func WithLoggerContextFieldsResolver(resolver ...mon.ContextFieldsResolver) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			return logger.Option(mon.WithContextFieldsResolver(resolver...))
		})
	}
}

func WithLoggerHook(hook mon.LoggerHook) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			return logger.Option(mon.WithHook(hook))
		})
	}
}

func WithLoggerLevel(level string) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			return logger.Option(mon.WithLevel(level))
		})
	}
}

func WithLoggerMetricHook(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		metricHook := mon.NewMetricHook()
		return logger.Option(mon.WithHook(metricHook))
	})
}

func WithLoggerSentryHook(extraProvider ...mon.SentryExtraProvider) Option {
	return func(app *App) {
		app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			var err error

			env := config.GetString("env")
			sentryHook := mon.NewSentryHook(env)

			for _, p := range extraProvider {
				sentryHook, err = p(config, sentryHook)

				if err != nil {
					return errors.Wrap(err, "can not configure LoggerSentryHook")
				}
			}

			return logger.Option(mon.WithHook(sentryHook))
		})
	}
}

func WithLoggerSettingsFromConfig(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		settings := &loggerSettings{}
		config.UnmarshalKey("mon.logger", settings)

		loggerOptions := []mon.LoggerOption{
			mon.WithLevel(settings.Level),
		}

		loggerOptions, err := getLoggerHandlersFromConfigWithFallback(config, loggerOptions)
		if err != nil {
			return err
		}

		return logger.Option(loggerOptions...)
	})
}

func getLoggerHandlersFromConfigWithFallback(config cfg.GosoConf, loggerOptions []mon.LoggerOption) ([]mon.LoggerOption, error) {
	handlers, err := getLoggerHandlersWithDefaultStdout(config)
	if err != nil {
		return nil, err
	}

	for handlerName, handler := range handlers {
		factory, ok := mon.HandlerFactories[handler.Name]
		if !ok {
			return nil, fmt.Errorf("unknown logger handler '%s'", handler.Name)
		}

		logHandler, err := factory(config, handlerName)
		if err != nil {
			return nil, err
		}

		loggerOptions = append(loggerOptions, mon.WithHandler(logHandler))
	}

	return loggerOptions, nil
}

func getLoggerHandlersWithDefaultStdout(config cfg.GosoConf) (map[string]loggerHandler, error) {
	key := "mon.logger.handler"
	if !config.IsSet(key) {
		err := config.Option(cfg.WithConfigMap(map[string]interface{}{
			"mon": map[string]interface{}{
				"logger": map[string]interface{}{
					"handler": map[string]interface{}{
						"stdout": map[string]interface{}{
							"type":            "stdout",
							"levels":          []string{"info", "warn", "error", "fatal", "panic"},
							"outputFormat":    mon.FormatConsole,
							"timestampFormat": time.RFC3339,
						},
					},
				},
			},
		}))

		if err != nil {
			return nil, err
		}
	}

	handlers := map[string]loggerHandler{}
	config.UnmarshalKey(key, &handlers)

	return handlers, nil
}

func WithLoggerTagsFromConfig(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		settings := &loggerSettings{}
		config.UnmarshalKey("mon.logger", settings)

		return logger.Option(mon.WithTags(settings.Tags))
	})
}

func WithMetricDaemon(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.GosoKernel) error {
		kernel.Add("metric", mon.ProvideCwDaemon())
		return nil
	})
}

func WithProducerDaemon(app *App) {
	app.addKernelOption(func(config cfg.GosoConf, kernel kernel.GosoKernel) error {
		kernel.AddFactory(stream.ProducerDaemonFactory)
		return nil
	})
}

func WithTracing(app *App) {
	app.addLoggerOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		tracingHook := tracing.NewLoggerErrorHook()

		options := []mon.LoggerOption{
			mon.WithHook(tracingHook),
			mon.WithContextFieldsResolver(tracing.ContextTraceFieldsResolver),
		}

		return logger.Option(options...)
	})

	app.addSetupOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
		strategy := tracing.NewTraceIdErrorWarningStrategy(logger)
		stream.AddDefaultEncodeHandler(tracing.NewMessageWithTraceEncoder(strategy))

		return nil
	})
}

func WithUTCClock(useUTC bool) Option {
	return func(app *App) {
		app.addSetupOption(func(config cfg.GosoConf, logger mon.GosoLog) error {
			clock.WithUseUTC(useUTC)

			return nil
		})
	}
}
